package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpServer struct {
	server *mcp.Server
	log    *log.Logger

	setTopicFn          func(context.Context, string) error
	askQuestionFn       func(context.Context, string) (string, error)
	planGetCurrentDirFn func() (string, error)
	planRequestDirFn    func() (planAllocation, error)
	planRemoveThreadFn  func(string) error
	planClearCurrentFn  func() error
	planCommitFn        func(context.Context, string) (int, error)
}

func newMCPServer() *mcpServer {
	logFile, err := os.OpenFile("/tmp/flowgentic-agentctl-mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		logFile = os.Stderr
	}

	s := &mcpServer{
		log:                 log.New(logFile, "", 0),
		setTopicFn:          runSetTopic,
		askQuestionFn:       mockAskQuestion,
		planGetCurrentDirFn: planGetCurrentDir,
		planRequestDirFn:    planRequestThreadDir,
		planRemoveThreadFn:  planRemoveThread,
		planClearCurrentFn:  planClearCurrent,
		planCommitFn:        planCommit,
	}
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "agentctl",
		Version: "1.0.0",
	}, nil)
	s.registerTools()
	return s
}

func (s *mcpServer) Run(ctx context.Context) error {
	s.logf("mcp serve start")

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	modeCh := make(chan bool, 1) // true = content-length framing, false = newline-delimited
	errCh := make(chan error, 2)

	go func() {
		errCh <- adaptInput(os.Stdin, inW, modeCh)
	}()
	go func() {
		headerMode := <-modeCh
		if headerMode {
			s.logf("framing detected: content-length")
		} else {
			s.logf("framing detected: newline")
		}
		errCh <- adaptOutput(outR, os.Stdout, headerMode)
	}()

	runErr := s.server.Run(ctx, &mcp.IOTransport{
		Reader: inR,
		Writer: outW,
	})
	if runErr != nil {
		s.logf("server run error: %v", runErr)
	}
	_ = inR.Close()
	_ = outW.Close()
	_ = outR.Close()
	_ = inW.Close()

	select {
	case adaptErr := <-errCh:
		if runErr != nil {
			return runErr
		}
		if adaptErr != nil && adaptErr != io.EOF {
			s.logf("adapter error: %v", adaptErr)
			return adaptErr
		}
	default:
	}
	return runErr
}

type setTopicArgs struct {
	Topic string `json:"topic" jsonschema:"A concise topic for this run (max 100 characters)."`
}

type setTopicResult struct {
	Topic string `json:"topic"`
}

type askQuestionArgs struct {
	Question string `json:"question" jsonschema:"Short clarifying question for plan scope."`
}

type askQuestionResult struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Mocked   bool   `json:"mocked"`
}

type planGetCurrentDirResult struct {
	PlanDir string `json:"plan_dir"`
}

type planRequestThreadDirResult struct {
	ThreadID string `json:"thread_id"`
	PlanDir  string `json:"plan_dir"`
}

type planRemoveThreadArgs struct {
	ThreadID string `json:"thread_id" jsonschema:"Additional thread ID to remove."`
}

type planRemoveThreadResult struct {
	ThreadID string `json:"thread_id"`
}

type planClearCurrentResult struct {
	Cleared bool `json:"cleared"`
}

type planCommitResult struct {
	SubmittedPlans int `json:"submitted_plans"`
}

func (s *mcpServer) registerTools() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "set_topic",
		Description: "Set a short topic for the current thread.",
	}, s.handleSetTopic)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "ask_question",
		Description: "Ask a clarifying question and receive a mocked answer for planning.",
	}, s.handleAskQuestion)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "plan_get_current_dir",
		Description: "Get the current thread plan directory path.",
	}, s.handlePlanGetCurrentDir)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "plan_request_thread_dir",
		Description: "Allocate an additional thread plan directory.",
	}, s.handlePlanRequestThreadDir)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "plan_remove_thread",
		Description: "Remove an allocated additional thread plan directory by thread ID.",
	}, s.handlePlanRemoveThread)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "plan_clear_current",
		Description: "Clear all files in the current thread plan directory.",
	}, s.handlePlanClearCurrent)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "plan_commit",
		Description: "Validate and submit all allocated plan directories.",
	}, s.handlePlanCommit)
}

func (s *mcpServer) handleSetTopic(ctx context.Context, _ *mcp.CallToolRequest, args setTopicArgs) (*mcp.CallToolResult, setTopicResult, error) {
	s.logf("tool call: set_topic")
	topic := strings.TrimSpace(args.Topic)
	if topic == "" {
		s.logf("tool call: set_topic failed: empty topic")
		return nil, setTopicResult{}, fmt.Errorf("topic is required")
	}
	if len([]rune(topic)) > 100 {
		s.logf("tool call: set_topic failed: topic too long (%d runes)", len([]rune(topic)))
		return nil, setTopicResult{}, fmt.Errorf("topic must be at most 100 characters")
	}
	if err := s.setTopicFn(ctx, topic); err != nil {
		s.logf("tool call: set_topic failed: %v", err)
		return nil, setTopicResult{}, err
	}
	s.logf("tool call: set_topic completed: %q", topic)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Topic set successfully"},
		},
	}, setTopicResult{Topic: topic}, nil
}

func (s *mcpServer) handleAskQuestion(ctx context.Context, _ *mcp.CallToolRequest, args askQuestionArgs) (*mcp.CallToolResult, askQuestionResult, error) {
	s.logf("tool call: ask_question")
	question := strings.TrimSpace(args.Question)
	if question == "" {
		return nil, askQuestionResult{}, fmt.Errorf("question is required")
	}
	answer, err := s.askQuestionFn(ctx, question)
	if err != nil {
		return nil, askQuestionResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: answer},
		},
	}, askQuestionResult{
		Question: question,
		Answer:   answer,
		Mocked:   true,
	}, nil
}

func mockAskQuestion(_ context.Context, _ string) (string, error) {
	return "Mock response: no live user input is available in this mode; proceed with sensible defaults.", nil
}

func (s *mcpServer) handlePlanGetCurrentDir(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, planGetCurrentDirResult, error) {
	s.logf("tool call: plan_get_current_dir")
	dir, err := s.planGetCurrentDirFn()
	if err != nil {
		return nil, planGetCurrentDirResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: dir},
		},
	}, planGetCurrentDirResult{PlanDir: dir}, nil
}

func (s *mcpServer) handlePlanRequestThreadDir(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, planRequestThreadDirResult, error) {
	s.logf("tool call: plan_request_thread_dir")
	a, err := s.planRequestDirFn()
	if err != nil {
		return nil, planRequestThreadDirResult{}, err
	}
	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Allocated additional plan directory"},
			},
		}, planRequestThreadDirResult{
			ThreadID: a.ThreadID,
			PlanDir:  a.PlanDir,
		}, nil
}

func (s *mcpServer) handlePlanRemoveThread(_ context.Context, _ *mcp.CallToolRequest, args planRemoveThreadArgs) (*mcp.CallToolResult, planRemoveThreadResult, error) {
	s.logf("tool call: plan_remove_thread")
	threadID := strings.TrimSpace(args.ThreadID)
	if threadID == "" {
		return nil, planRemoveThreadResult{}, fmt.Errorf("thread_id is required")
	}
	if err := s.planRemoveThreadFn(threadID); err != nil {
		return nil, planRemoveThreadResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Removed thread " + threadID},
		},
	}, planRemoveThreadResult{ThreadID: threadID}, nil
}

func (s *mcpServer) handlePlanClearCurrent(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, planClearCurrentResult, error) {
	s.logf("tool call: plan_clear_current")
	if err := s.planClearCurrentFn(); err != nil {
		return nil, planClearCurrentResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Cleared current plan content"},
		},
	}, planClearCurrentResult{Cleared: true}, nil
}

func (s *mcpServer) handlePlanCommit(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, planCommitResult, error) {
	s.logf("tool call: plan_commit")
	submitted, err := s.planCommitFn(ctx, os.Getenv("AGENTCTL_AGENT"))
	if err != nil {
		return nil, planCommitResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Plan submitted successfully (%d plan dirs)", submitted)},
		},
	}, planCommitResult{SubmittedPlans: submitted}, nil
}

func (s *mcpServer) logf(format string, args ...any) {
	if s.log == nil {
		return
	}
	s.log.Printf("%s %s", time.Now().Format(time.RFC3339Nano), fmt.Sprintf(format, args...))
}

func adaptInput(src io.Reader, dst *io.PipeWriter, modeCh chan<- bool) error {
	defer close(modeCh)
	defer func() { _ = dst.Close() }()

	br := bufio.NewReader(src)

	line, err := br.ReadString('\n')
	if err != nil {
		return err
	}

	headerMode := strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "content-length:")
	modeCh <- headerMode

	if headerMode {
		if err := forwardHeaderFramedInput(line, br, dst); err != nil {
			return err
		}
		return nil
	}

	if _, err := io.WriteString(dst, line); err != nil {
		return err
	}
	_, err = io.Copy(dst, br)
	return err
}

func forwardHeaderFramedInput(firstLine string, br *bufio.Reader, dst *io.PipeWriter) error {
	line := firstLine
	for {
		contentLength, ok, err := parseContentLengthLine(line)
		if err != nil {
			return err
		}
		if !ok {
			line, err = br.ReadString('\n')
			if err != nil {
				return err
			}
			continue
		}

		for {
			h, err := br.ReadString('\n')
			if err != nil {
				return err
			}
			if strings.TrimSpace(h) == "" {
				break
			}
		}

		payload := make([]byte, contentLength)
		if _, err := io.ReadFull(br, payload); err != nil {
			return err
		}
		if _, err := dst.Write(payload); err != nil {
			return err
		}
		if _, err := dst.Write([]byte{'\n'}); err != nil {
			return err
		}

		line, err = br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func parseContentLengthLine(line string) (int, bool, error) {
	parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
	if len(parts) != 2 {
		return 0, false, nil
	}
	if strings.ToLower(strings.TrimSpace(parts[0])) != "content-length" {
		return 0, false, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || n < 0 {
		return 0, false, fmt.Errorf("invalid Content-Length")
	}
	return n, true, nil
}

func adaptOutput(src io.Reader, dst io.Writer, headerMode bool) error {
	br := bufio.NewReader(src)
	bw := bufio.NewWriter(dst)
	defer bw.Flush()

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if headerMode {
			if _, err := fmt.Fprintf(bw, "Content-Length: %d\r\n\r\n%s", len(trimmed), trimmed); err != nil {
				return err
			}
		} else {
			if _, err := bw.WriteString(trimmed + "\n"); err != nil {
				return err
			}
		}
		if err := bw.Flush(); err != nil {
			return err
		}
	}
}
