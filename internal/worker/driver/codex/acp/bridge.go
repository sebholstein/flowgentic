package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/sebastianm/flowgentic/internal/worker/procutil"
)

// jsonrpcRequest is a JSON-RPC 2.0 request or notification.
type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

type jsonrpcNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// bridge manages the Codex app-server subprocess and JSON-RPC communication.
type bridge struct {
	log     *slog.Logger
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdinMu sync.Mutex

	modelState        *acpsdk.SessionModelState
	availableCommands []acpsdk.AvailableCommand

	nextID    atomic.Int64
	pending   map[int64]chan jsonrpcResponse
	pendingMu sync.Mutex

	dispatch func(threadID string, method string, params json.RawMessage, serverRequestID *int64)

	done chan struct{}
}

func newBridge(log *slog.Logger, dispatch func(threadID string, method string, params json.RawMessage, serverRequestID *int64)) *bridge {
	return &bridge{
		log:      log,
		pending:  make(map[int64]chan jsonrpcResponse),
		dispatch: dispatch,
		done:     make(chan struct{}),
	}
}

func (b *bridge) start(ctx context.Context, envVars map[string]string) error {
	cmd := exec.CommandContext(ctx, "codex", "app-server")
	cmd.Env = driver.BuildEnv(envVars)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := procutil.StartWithCleanup(cmd); err != nil {
		return fmt.Errorf("start codex app-server: %w", err)
	}

	b.cmd = cmd
	b.stdin = stdin

	go b.readLoop(stdout)
	go b.readStderrLoop(stderr)

	// Handshake.
	initResult, err := b.sendRequest("initialize", map[string]any{
		"clientInfo":   map[string]string{"name": "flowgentic", "version": "1.0.0"},
		"capabilities": map[string]bool{"experimentalApi": true},
	})
	if err != nil {
		b.close()
		return fmt.Errorf("initialize handshake: %w", err)
	}
	b.modelState = parseModelState(initResult)
	b.availableCommands = parseAvailableCommands(initResult)

	b.sendNotification("initialized", nil)
	return nil
}

func (b *bridge) sendRequest(method string, params any) (json.RawMessage, error) {
	id := b.nextID.Add(1)
	ch := make(chan jsonrpcResponse, 1)

	b.pendingMu.Lock()
	b.pending[id] = ch
	b.pendingMu.Unlock()

	req := jsonrpcRequest{JSONRPC: "2.0", ID: &id, Method: method, Params: params}
	if err := b.writeJSON(req); err != nil {
		b.pendingMu.Lock()
		delete(b.pending, id)
		b.pendingMu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-b.done:
		return nil, fmt.Errorf("app-server closed before response for %s (id=%d)", method, id)
	}
}

func (b *bridge) sendNotification(method string, params any) {
	_ = b.writeJSON(jsonrpcRequest{JSONRPC: "2.0", Method: method, Params: params})
}

func (b *bridge) respondToServerRequest(id int64, result any) {
	type resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Result  any    `json:"result"`
	}
	_ = b.writeJSON(resp{JSONRPC: "2.0", ID: id, Result: result})
}

func (b *bridge) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal JSON-RPC: %w", err)
	}
	data = append(data, '\n')
	b.stdinMu.Lock()
	defer b.stdinMu.Unlock()
	_, err = b.stdin.Write(data)
	return err
}

func (b *bridge) readLoop(r io.Reader) {
	defer close(b.done)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg jsonrpcNotification
		if err := json.Unmarshal(line, &msg); err != nil {
			b.log.Warn("invalid JSON from app-server", "error", err)
			continue
		}

		if msg.Method == "" {
			var resp jsonrpcResponse
			if json.Unmarshal(line, &resp) == nil && resp.ID != nil {
				b.pendingMu.Lock()
				ch, ok := b.pending[*resp.ID]
				if ok {
					delete(b.pending, *resp.ID)
				}
				b.pendingMu.Unlock()
				if ok {
					ch <- resp
				}
			}
			continue
		}

		threadID := extractThreadID(msg.Params)
		b.dispatch(threadID, msg.Method, msg.Params, msg.ID)
	}
}

func (b *bridge) readStderrLoop(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		b.log.Debug("codex stderr", "line", line)
	}
}

func (b *bridge) threadStart(model, cwd, systemPrompt, sessionMode string, mcpServers []acpsdk.McpServer) (string, error) {
	policy := "on-failure"
	if sessionMode == "code" {
		policy = "never"
	}
	params := map[string]any{
		"cwd":            cwd,
		"approvalPolicy": policy,
	}
	if cfg := codexMCPServers(mcpServers); len(cfg) > 0 {
		params["config"] = map[string]any{
			"mcp_servers": cfg,
		}
	}
	if model != "" {
		params["model"] = model
	}
	if systemPrompt != "" {
		params["developerInstructions"] = systemPrompt
	}

	result, err := b.sendRequest("thread/start", params)
	if err != nil {
		return "", err
	}
	var res struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		Conversation struct {
			ID string `json:"id"`
		} `json:"conversation"`
		ThreadID       string `json:"threadId"`
		ConversationID string `json:"conversationId"`
	}
	if err := json.Unmarshal(result, &res); err != nil {
		return "", fmt.Errorf("parse thread/start: %w", err)
	}
	switch {
	case res.Thread.ID != "":
		return res.Thread.ID, nil
	case res.Conversation.ID != "":
		return res.Conversation.ID, nil
	case res.ThreadID != "":
		return res.ThreadID, nil
	case res.ConversationID != "":
		return res.ConversationID, nil
	default:
		return "", fmt.Errorf("parse thread/start: missing thread/conversation id")
	}
}

func (b *bridge) turnStart(threadID, prompt, cwd, sessionMode string) (string, error) {
	sp := map[string]any{
		"type":          "workspaceWrite",
		"writableRoots": []string{cwd},
		"networkAccess": true,
	}
	if sessionMode == "code" {
		sp = map[string]any{
			"type":          "dangerFullAccess",
			"networkAccess": true,
		}
	}

	result, err := b.sendRequest("turn/start", map[string]any{
		"threadId":      threadID,
		"input":         []map[string]string{{"type": "text", "text": prompt}},
		"sandboxPolicy": sp,
	})
	if err != nil {
		return "", err
	}
	var res struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(result, &res); err != nil {
		return "", fmt.Errorf("parse turn/start: %w", err)
	}
	return res.Turn.ID, nil
}

func (b *bridge) turnInterrupt(threadID, turnID string) error {
	_, err := b.sendRequest("turn/interrupt", map[string]string{
		"threadId": threadID,
		"turnId":   turnID,
	})
	return err
}

func (b *bridge) close() {
	if b.stdin != nil {
		_ = b.stdin.Close()
	}
	if b.cmd != nil && b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
		_ = b.cmd.Wait()
	}
}

func (b *bridge) doneChan() <-chan struct{} {
	return b.done
}

func (b *bridge) modelSnapshot() *acpsdk.SessionModelState {
	if b.modelState == nil {
		return nil
	}
	cloned := *b.modelState
	cloned.AvailableModels = append([]acpsdk.ModelInfo(nil), b.modelState.AvailableModels...)
	return &cloned
}

func (b *bridge) availableCommandsSnapshot() []acpsdk.AvailableCommand {
	return append([]acpsdk.AvailableCommand(nil), b.availableCommands...)
}

func (b *bridge) request(method string, params any) (json.RawMessage, error) {
	return b.sendRequest(method, params)
}

func extractThreadID(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}
	var p struct {
		ThreadID       string `json:"threadId"`
		ConversationID string `json:"conversationId"`
		Thread         struct {
			ID string `json:"id"`
		} `json:"thread"`
		Conversation struct {
			ID string `json:"id"`
		} `json:"conversation"`
	}
	_ = json.Unmarshal(params, &p)
	switch {
	case p.ThreadID != "":
		return p.ThreadID
	case p.ConversationID != "":
		return p.ConversationID
	case p.Thread.ID != "":
		return p.Thread.ID
	case p.Conversation.ID != "":
		return p.Conversation.ID
	default:
		return ""
	}
}

func parseModelState(raw json.RawMessage) *acpsdk.SessionModelState {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}

	var currentModel string

	type modelEntry struct {
		id          string
		displayName string
		description string
	}
	var modelEntries []modelEntry

	appendModel := func(v any) {
		switch m := v.(type) {
		case string:
			if m != "" {
				modelEntries = append(modelEntries, modelEntry{id: m})
			}
		case map[string]any:
			var id string
			for _, key := range []string{"id", "model", "modelId", "name"} {
				if mv, ok := m[key].(string); ok && mv != "" {
					id = mv
					break
				}
			}
			if id == "" {
				return
			}
			e := modelEntry{id: id}
			for _, key := range []string{"displayName", "display_name", "name"} {
				if mv, ok := m[key].(string); ok && mv != "" {
					e.displayName = mv
					break
				}
			}
			if mv, ok := m["description"].(string); ok {
				e.description = mv
			}
			modelEntries = append(modelEntries, e)
		}
	}

	if v, ok := data["availableModels"]; ok {
		if list, ok := v.([]any); ok {
			for _, item := range list {
				appendModel(item)
			}
		}
	}
	if v, ok := data["models"]; ok {
		switch typed := v.(type) {
		case []any:
			for _, item := range typed {
				appendModel(item)
			}
		case map[string]any:
			for _, key := range []string{"available", "availableModels"} {
				if list, ok := typed[key].([]any); ok {
					for _, item := range list {
						appendModel(item)
					}
				}
			}
			for _, key := range []string{"current", "currentModel", "currentModelId", "default", "defaultModel", "defaultModelId"} {
				if s, ok := typed[key].(string); ok && s != "" {
					currentModel = s
					break
				}
			}
		}
	}

	if currentModel == "" {
		for _, key := range []string{"currentModel", "currentModelId", "defaultModel", "defaultModelId", "model"} {
			if s, ok := data[key].(string); ok && s != "" {
				currentModel = s
				break
			}
		}
	}

	uniq := make([]modelEntry, 0, len(modelEntries))
	seen := map[string]struct{}{}
	for _, e := range modelEntries {
		if _, ok := seen[e.id]; ok {
			continue
		}
		seen[e.id] = struct{}{}
		uniq = append(uniq, e)
	}
	if len(uniq) == 0 || currentModel == "" {
		return nil
	}

	state := &acpsdk.SessionModelState{
		AvailableModels: make([]acpsdk.ModelInfo, 0, len(uniq)),
		CurrentModelId:  acpsdk.ModelId(currentModel),
	}
	for _, e := range uniq {
		info := acpsdk.ModelInfo{
			ModelId: acpsdk.ModelId(e.id),
			Name:    e.displayName,
		}
		if e.description != "" {
			info.Description = &e.description
		}
		state.AvailableModels = append(state.AvailableModels, info)
	}
	return state
}

type commandEnvelope struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Summary     string `json:"summary"`
}

type initializeResultEnvelope struct {
	AvailableCommands json.RawMessage `json:"availableCommands"`
	AvailableSnake    json.RawMessage `json:"available_commands"`
	Commands          json.RawMessage `json:"commands"`
	Skills            json.RawMessage `json:"skills"`
	Data              json.RawMessage `json:"data"`
	Message           json.RawMessage `json:"message"`
	Capabilities      struct {
		AvailableCommands json.RawMessage `json:"availableCommands"`
		AvailableSnake    json.RawMessage `json:"available_commands"`
		Commands          json.RawMessage `json:"commands"`
		Skills            json.RawMessage `json:"skills"`
	} `json:"capabilities"`
}

func parseAvailableCommands(raw json.RawMessage) []acpsdk.AvailableCommand {
	var env initializeResultEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil
	}

	var out []acpsdk.AvailableCommand
	addRaw := func(candidate json.RawMessage) {
		out = appendDedupCommands(out, parseCommandArray(candidate)...)
	}
	addRaw(env.AvailableCommands)
	addRaw(env.AvailableSnake)
	addRaw(env.Commands)
	addRaw(env.Capabilities.AvailableCommands)
	addRaw(env.Capabilities.AvailableSnake)
	addRaw(env.Capabilities.Commands)
	addRaw(env.Capabilities.Skills)
	addRaw(env.Skills)
	addRaw(env.Data)
	addRaw(env.Message)

	if len(out) > 0 {
		return out
	}

	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}
	return parseAvailableCommandsFallback(root)
}

func parseAvailableCommandsFallback(root map[string]any) []acpsdk.AvailableCommand {
	var out []acpsdk.AvailableCommand
	addAny := func(v any) {
		out = appendDedupCommands(out, parseCommandAny(v)...)
	}

	addCandidateMap := func(m map[string]any) {
		for _, key := range []string{"availableCommands", "available_commands", "commands", "skills"} {
			addAny(m[key])
		}
		if inner, ok := m["capabilities"].(map[string]any); ok {
			for _, key := range []string{"availableCommands", "available_commands", "commands", "skills"} {
				addAny(inner[key])
			}
		}
		if initialMessages, ok := m["initialMessages"].([]any); ok {
			for _, msg := range initialMessages {
				msgMap, ok := msg.(map[string]any)
				if !ok {
					continue
				}
				addAny(msgMap["skills"])
				addAny(msgMap["availableCommands"])
				addAny(msgMap["available_commands"])
			}
		}
	}

	addCandidateMap(root)
	for _, key := range []string{"data", "message"} {
		if nested, ok := root[key].(map[string]any); ok {
			addCandidateMap(nested)
		}
	}

	return out
}

func parseCommandArray(raw json.RawMessage) []acpsdk.AvailableCommand {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var entries []commandEnvelope
	if err := json.Unmarshal(raw, &entries); err == nil {
		out := make([]acpsdk.AvailableCommand, 0, len(entries))
		for _, entry := range entries {
			if cmd, ok := normalizeAvailableCommand(entry.Name, entry.Command, entry.ID, entry.Slug, entry.Title, entry.Description, entry.Summary); ok {
				out = appendDedupCommands(out, cmd)
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	var asAny any
	if err := json.Unmarshal(raw, &asAny); err != nil {
		return nil
	}
	return parseCommandAny(asAny)
}

func parseCommandAny(v any) []acpsdk.AvailableCommand {
	switch typed := v.(type) {
	case []any:
		var out []acpsdk.AvailableCommand
		for _, item := range typed {
			out = appendDedupCommands(out, parseCommandAny(item)...)
		}
		return out
	case map[string]any:
		var out []acpsdk.AvailableCommand
		if cmd, ok := normalizeAvailableCommand(
			anyString(typed["name"]),
			anyString(typed["command"]),
			anyString(typed["id"]),
			anyString(typed["slug"]),
			anyString(typed["title"]),
			anyString(typed["description"]),
			anyString(typed["summary"]),
		); ok {
			out = append(out, cmd)
		}
		for _, key := range []string{"availableCommands", "available_commands", "commands", "skills", "data", "message", "capabilities"} {
			out = appendDedupCommands(out, parseCommandAny(typed[key])...)
		}
		return out
	case string:
		if cmd, ok := normalizeAvailableCommand(typed, "", "", "", "", "", ""); ok {
			return []acpsdk.AvailableCommand{cmd}
		}
	}
	return nil
}

func normalizeAvailableCommand(name, command, id, slug, title, description, summary string) (acpsdk.AvailableCommand, bool) {
	for _, candidate := range []string{name, command, id, slug, title} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		desc := strings.TrimSpace(description)
		if desc == "" {
			desc = strings.TrimSpace(summary)
		}
		return acpsdk.AvailableCommand{
			Name:        candidate,
			Description: desc,
		}, true
	}
	return acpsdk.AvailableCommand{}, false
}

func appendDedupCommands(dst []acpsdk.AvailableCommand, src ...acpsdk.AvailableCommand) []acpsdk.AvailableCommand {
	if len(src) == 0 {
		return dst
	}
	seen := make(map[string]struct{}, len(dst))
	for _, cmd := range dst {
		seen[cmd.Name] = struct{}{}
	}
	for _, cmd := range src {
		name := strings.TrimSpace(cmd.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		dst = append(dst, acpsdk.AvailableCommand{
			Name:        name,
			Description: strings.TrimSpace(cmd.Description),
		})
	}
	return dst
}

func anyString(v any) string {
	s, _ := v.(string)
	return s
}

func codexMCPServers(servers []acpsdk.McpServer) map[string]any {
	out := make(map[string]any, len(servers))
	nextUnnamed := 1
	serverName := func(name string) string {
		if name != "" {
			return name
		}
		n := fmt.Sprintf("mcp-%d", nextUnnamed)
		nextUnnamed++
		return n
	}
	for _, server := range servers {
		switch {
		case server.Stdio != nil:
			env := map[string]string{}
			for _, kv := range server.Stdio.Env {
				if kv.Name == "" {
					continue
				}
				env[kv.Name] = kv.Value
			}
			args := make([]string, 0, len(server.Stdio.Args))
			for _, arg := range server.Stdio.Args {
				if strings.TrimSpace(arg) == "" {
					continue
				}
				args = append(args, arg)
			}
			out[serverName(server.Stdio.Name)] = map[string]any{
				"type":    "stdio",
				"command": server.Stdio.Command,
				"args":    args,
				"env":     env,
			}
		case server.Sse != nil:
			headers := map[string]string{}
			for _, h := range server.Sse.Headers {
				if h.Name == "" {
					continue
				}
				headers[h.Name] = h.Value
			}
			out[serverName(server.Sse.Name)] = map[string]any{
				"type":    "sse",
				"url":     server.Sse.Url,
				"headers": headers,
			}
		case server.Http != nil:
			headers := map[string]string{}
			for _, h := range server.Http.Headers {
				if h.Name == "" {
					continue
				}
				headers[h.Name] = h.Value
			}
			out[serverName(server.Http.Name)] = map[string]any{
				"type":    "http",
				"url":     server.Http.Url,
				"headers": headers,
			}
		}
	}
	return out
}
