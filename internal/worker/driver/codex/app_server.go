package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/sebastianm/flowgentic/internal/worker/procutil"
)

// jsonrpcRequest is a JSON-RPC 2.0 request or notification.
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64      `json:"id,omitempty"` // nil for notifications
	Method  string      `json:"method"`
	Params  any `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError is a JSON-RPC 2.0 error object.
type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// jsonrpcNotification is an incoming JSON-RPC 2.0 notification (no id)
// or a server-initiated request (has id, expecting a response).
type jsonrpcNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// appServer manages a shared `codex app-server` subprocess.
type appServer struct {
	log    *slog.Logger
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdinMu sync.Mutex // protects writes to stdin

	nextID  atomic.Int64
	pending   map[int64]chan jsonrpcResponse
	pendingMu sync.Mutex

	// dispatch routes notifications to sessions by threadID.
	dispatch func(threadID string, method string, params json.RawMessage, serverRequestID *int64)

	done chan struct{} // closed when readLoop exits
}

// initializeParams are sent with the initialize request.
type initializeParams struct {
	ClientInfo   clientInfo   `json:"clientInfo"`
	Capabilities capabilities `json:"capabilities"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type capabilities struct {
	ExperimentalAPI bool `json:"experimentalApi"`
}

// threadStartParams are sent with thread/start.
type threadStartParams struct {
	Model                 string `json:"model,omitempty"`
	Cwd                   string `json:"cwd,omitempty"`
	ApprovalPolicy        string `json:"approvalPolicy,omitempty"`
	DeveloperInstructions string `json:"developerInstructions,omitempty"`
}

// threadStartResult is the result of thread/start.
type threadStartResult struct {
	Thread struct {
		ID string `json:"id"`
	} `json:"thread"`
}

// turnStartParams are sent with turn/start.
type turnStartParams struct {
	ThreadID string    `json:"threadId"`
	Input    []turnInput `json:"input"`
}

type turnInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// turnInterruptParams are sent with turn/interrupt.
type turnInterruptParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

func newAppServer(log *slog.Logger, dispatch func(threadID string, method string, params json.RawMessage, serverRequestID *int64)) *appServer {
	return &appServer{
		log:      log,
		pending:  make(map[int64]chan jsonrpcResponse),
		dispatch: dispatch,
		done:     make(chan struct{}),
	}
}

// start spawns the codex app-server process and performs the initialize handshake.
func (s *appServer) start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "codex", "app-server")
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := procutil.StartWithCleanup(cmd); err != nil {
		return fmt.Errorf("start codex app-server: %w", err)
	}

	s.cmd = cmd
	s.stdin = stdin

	go s.readLoop(stdout)

	// Perform initialize handshake.
	_, err = s.sendRequest("initialize", initializeParams{
		ClientInfo: clientInfo{
			Name:    "flowgentic",
			Version: "0.1.0",
		},
		Capabilities: capabilities{
			ExperimentalAPI: true,
		},
	})
	if err != nil {
		s.close()
		return fmt.Errorf("initialize handshake: %w", err)
	}

	// Send initialized notification.
	s.sendNotification("initialized", nil)

	return nil
}

// sendRequest sends a JSON-RPC request and blocks until the response arrives.
func (s *appServer) sendRequest(method string, params any) (json.RawMessage, error) {
	id := s.nextID.Add(1)
	ch := make(chan jsonrpcResponse, 1)

	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}

	if err := s.writeJSON(req); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return nil, err
	}

	// Wait for response or server shutdown.
	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-s.done:
		return nil, fmt.Errorf("app-server closed before response for %s (id=%d)", method, id)
	}
}

// sendNotification sends a JSON-RPC notification (no id, no response expected).
func (s *appServer) sendNotification(method string, params any) {
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	_ = s.writeJSON(req)
}

// respondToServerRequest sends a JSON-RPC response to a server-initiated request.
func (s *appServer) respondToServerRequest(id int64, result any) {
	type jsonrpcResponseOut struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Result  any    `json:"result"`
	}
	resp := jsonrpcResponseOut{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	_ = s.writeJSON(resp)
}

func (s *appServer) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal JSON-RPC: %w", err)
	}
	data = append(data, '\n')

	s.stdinMu.Lock()
	defer s.stdinMu.Unlock()

	_, err = s.stdin.Write(data)
	return err
}

// readLoop reads JSONL from stdout and dispatches responses and notifications.
func (s *appServer) readLoop(r io.Reader) {
	defer close(s.done)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Try to parse as a response (has "id" and either "result" or "error").
		var msg jsonrpcNotification
		if err := json.Unmarshal(line, &msg); err != nil {
			s.log.Warn("invalid JSON from app-server", "error", err)
			continue
		}

		if msg.Method == "" {
			// No method = this is a response to one of our requests.
			var resp jsonrpcResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				continue
			}
			if resp.ID != nil {
				s.pendingMu.Lock()
				ch, ok := s.pending[*resp.ID]
				if ok {
					delete(s.pending, *resp.ID)
				}
				s.pendingMu.Unlock()
				if ok {
					ch <- resp
				}
			}
			continue
		}

		// It's a notification or server-initiated request.
		// Extract threadId from params for routing.
		threadID := extractThreadID(msg.Params)
		s.dispatch(threadID, msg.Method, msg.Params, msg.ID)
	}
}

// threadStart creates a new thread on the app-server.
func (s *appServer) threadStart(model, cwd, systemPrompt string, yolo bool) (string, error) {
	policy := "never"
	if !yolo {
		policy = "on-failure"
	}

	params := threadStartParams{
		Model:          model,
		Cwd:            cwd,
		ApprovalPolicy: policy,
	}
	if systemPrompt != "" {
		params.DeveloperInstructions = systemPrompt
	}

	result, err := s.sendRequest("thread/start", params)
	if err != nil {
		return "", fmt.Errorf("thread/start: %w", err)
	}

	var res threadStartResult
	if err := json.Unmarshal(result, &res); err != nil {
		return "", fmt.Errorf("parse thread/start result: %w", err)
	}

	s.log.Debug("thread started", "threadID", res.Thread.ID)
	return res.Thread.ID, nil
}

// turnStart begins a new turn on a thread. Returns the turn ID.
func (s *appServer) turnStart(threadID, prompt string) (string, error) {
	params := turnStartParams{
		ThreadID: threadID,
		Input: []turnInput{
			{Type: "text", Text: prompt},
		},
	}

	result, err := s.sendRequest("turn/start", params)
	if err != nil {
		return "", fmt.Errorf("turn/start: %w", err)
	}

	var res struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(result, &res); err != nil {
		return "", fmt.Errorf("parse turn/start result: %w", err)
	}

	return res.Turn.ID, nil
}

// turnInterrupt interrupts the current turn on a thread.
func (s *appServer) turnInterrupt(threadID, turnID string) error {
	_, err := s.sendRequest("turn/interrupt", turnInterruptParams{
		ThreadID: threadID,
		TurnID:   turnID,
	})
	return err
}

// close shuts down the app-server process.
func (s *appServer) close() {
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
}

// extractThreadID pulls the threadId field from JSON-RPC params.
func extractThreadID(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}
	var p struct {
		ThreadID string `json:"threadId"`
	}
	_ = json.Unmarshal(params, &p)
	return p.ThreadID
}

// normalizeNotification converts an app-server JSON-RPC notification into driver Events.
func normalizeNotification(method string, params json.RawMessage) []driver.Event {
	now := currentTime()

	switch method {
	case "item/agentMessage/delta":
		var p struct {
			Delta string `json:"delta"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		return []driver.Event{{
			Type:      driver.EventTypeMessage,
			Timestamp: now,
			Agent:     agent,
			Text:      p.Delta,
			Delta:     true,
		}}

	case "item/reasoning/textDelta":
		var p struct {
			Delta string `json:"delta"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		return []driver.Event{{
			Type:      driver.EventTypeThinking,
			Timestamp: now,
			Agent:     agent,
			Text:      p.Delta,
			Delta:     true,
		}}

	case "item/started":
		var p struct {
			Item struct {
				ID      string `json:"id"`
				Type    string `json:"type"`
				Command string `json:"command,omitempty"`
			} `json:"item"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		switch p.Item.Type {
		case "commandExecution":
			return []driver.Event{{
				Type:      driver.EventTypeToolStart,
				Timestamp: now,
				Agent:     agent,
				ToolName:  "command_execution",
				ToolID:    p.Item.ID,
				Text:      p.Item.Command,
			}}
		}

	case "item/completed":
		var p struct {
			Item struct {
				ID               string `json:"id"`
				Type             string `json:"type"`
				Text             string `json:"text,omitempty"`
				Command          string `json:"command,omitempty"`
				AggregatedOutput string `json:"aggregatedOutput,omitempty"`
				ExitCode         *int   `json:"exitCode,omitempty"`
				Changes          []struct {
					Path string `json:"path"`
					Diff string `json:"diff"`
				} `json:"changes,omitempty"`
			} `json:"item"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		switch p.Item.Type {
		case "agentMessage":
			return []driver.Event{{
				Type:      driver.EventTypeMessage,
				Timestamp: now,
				Agent:     agent,
				Text:      p.Item.Text,
			}}
		case "reasoning":
			return []driver.Event{{
				Type:      driver.EventTypeThinking,
				Timestamp: now,
				Agent:     agent,
				Text:      p.Item.Text,
			}}
		case "commandExecution":
			isError := p.Item.ExitCode != nil && *p.Item.ExitCode != 0
			return []driver.Event{{
				Type:      driver.EventTypeToolResult,
				Timestamp: now,
				Agent:     agent,
				ToolID:    p.Item.ID,
				ToolError: isError,
				Text:      p.Item.AggregatedOutput,
			}}
		case "fileChange":
			var parts []string
			for _, c := range p.Item.Changes {
				if c.Diff != "" {
					parts = append(parts, c.Path+"\n"+c.Diff)
				} else {
					parts = append(parts, c.Path)
				}
			}
			text := ""
			if len(parts) > 0 {
				text = strings.Join(parts, "\n")
			}
			return []driver.Event{{
				Type:      driver.EventTypeToolResult,
				Timestamp: now,
				Agent:     agent,
				ToolID:    p.Item.ID,
				Text:      text,
			}}
		}

	case "turn/completed":
		return []driver.Event{{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
		}}

	case "error":
		return []driver.Event{{
			Type:      driver.EventTypeError,
			Timestamp: now,
			Agent:     agent,
			Error:     string(params),
			Raw:       params,
		}}
	}

	return nil
}
