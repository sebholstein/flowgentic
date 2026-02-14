package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	acp "github.com/coder/acp-go-sdk"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"

	// Adapter factories for in-process agents.
	claudeacp "github.com/sebastianm/flowgentic/internal/worker/driver/claude/acp"
	codexacp "github.com/sebastianm/flowgentic/internal/worker/driver/codex/acp"
)

// spinner shows a braille animation on stderr while the agent is working.
type spinner struct {
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{} // closed when goroutine exits
}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (s *spinner) start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	go func() {
		defer close(s.doneCh)
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				fmt.Fprint(os.Stderr, "\r\033[K") // clear spinner line
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, "\r\033[2m%s\033[0m", spinFrames[i%len(spinFrames)])
				i++
			}
		}
	}()
}

func (s *spinner) stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	done := s.doneCh
	s.mu.Unlock()
	<-done // wait for goroutine to clear the line
}

func main() {
	agent := flag.String("agent", "claude-code", "agent to use: claude-code, codex, opencode, gemini")
	cwd := flag.String("cwd", ".", "working directory for the agent")
	mode := flag.String("mode", "code", "session mode: ask, architect, code")
	model := flag.String("model", "", "model override")
	system := flag.String("system", "", "system prompt")
	flag.Parse()

	// Initial prompt from positional args or stdin.
	prompt := strings.Join(flag.Args(), " ")
	if prompt == "" && stdinHasData() {
		fmt.Fprintln(os.Stderr, "reading prompt from stdin...")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			prompt = scanner.Text()
		}
	}
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "no initial prompt provided; starting session and waiting for your first message")
	}

	// Resolve cwd to absolute path (required by some agents like Codex).
	absCwd, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid cwd: %v\n", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	config, err := agentConfig(*agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	drv := v2.NewDriver(log, config)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Channel for permission request IDs to auto-approve.
	permCh := make(chan string, 16)

	// Track per-tool-call state for coherent display.
	type toolState struct {
		title      string
		kind       string
		status     acp.ToolCallStatus
		inputShown bool
	}
	toolCalls := map[string]*toolState{}

	// Spinner shown while waiting for agent output.
	spin := &spinner{}

	// Track whether the last output to stdout ended mid-line (no trailing newline).
	needsNewline := false
	var updateCount uint64
	availableCommands := map[string]struct{}{}

	// Print raw ACP updates for full protocol visibility.
	printRaw := func(update acp.SessionUpdate) {
		if needsNewline {
			fmt.Println()
			needsNewline = false
		}
		if b, ok := tryMarshalChunkRaw(update); ok {
			fmt.Fprintf(os.Stderr, "\033[35m[acp raw] %s\033[0m\n", string(b))
			return
		}
		b, err := json.Marshal(update)
		if err != nil {
			fallback, ferr := json.Marshal(rawFallback(update, err))
			if ferr != nil {
				fmt.Fprintf(os.Stderr, "\033[35m[acp raw marshal error: %v]\033[0m\n", err)
				return
			}
			fmt.Fprintf(os.Stderr, "\033[35m[acp raw fallback] %s\033[0m\n", string(fallback))
			return
		}
		fmt.Fprintf(os.Stderr, "\033[35m[acp raw] %s\033[0m\n", string(b))
	}

	// ensureNewline prints a newline to stdout if the last agent text didn't end
	// with one, so subsequent stderr output (tool headers, status) starts clean.
	ensureNewline := func() {
		if needsNewline {
			fmt.Println()
			needsNewline = false
		}
	}

	onEvent := func(n acp.SessionNotification) {
		atomic.AddUint64(&updateCount, 1)
		u := n.Update
		printRaw(u)

		switch {
		case u.AvailableCommandsUpdate != nil:
			next := make(map[string]struct{}, len(u.AvailableCommandsUpdate.AvailableCommands))
			for _, cmd := range u.AvailableCommandsUpdate.AvailableCommands {
				if cmd.Name == "" {
					continue
				}
				next[cmd.Name] = struct{}{}
			}
			availableCommands = next

		case u.AgentMessageChunk != nil:
			if u.AgentMessageChunk.Content.Text != nil {
				spin.stop()
				text := u.AgentMessageChunk.Content.Text.Text
				fmt.Print(text)
				needsNewline = len(text) > 0 && text[len(text)-1] != '\n'
			}

		case u.AgentThoughtChunk != nil:
			if u.AgentThoughtChunk.Content.Text != nil {
				spin.stop()
				fmt.Fprintf(os.Stderr, "\033[2m%s\033[0m", u.AgentThoughtChunk.Content.Text.Text)
			}

		case u.ToolCall != nil:
			spin.stop()
			ensureNewline()
			tc := u.ToolCall
			id := string(tc.ToolCallId)

			// Check if this is a permission request — auto-approve it.
			if raw, ok := tc.RawInput.(map[string]any); ok {
				if _, isPerm := raw["_permissionRequest"]; isPerm {
					permCh <- id
					fmt.Fprintf(os.Stderr, "\033[33m[permission: %s → auto-approved]\033[0m\n", tc.Title)
					return
				}
			}

			st := &toolState{title: tc.Title, kind: string(tc.Kind), status: tc.Status}
			toolCalls[id] = st

			// Skip printing pending-only starts — they'll be shown when
			// in_progress arrives with input. Only print if we already
			// have input or a non-pending status.
			if tc.Status != acp.ToolCallStatusPending || tc.RawInput != nil {
				printToolHeader(tc.Title, string(tc.Status), string(tc.Kind))
				printLocations(tc.Locations)
				if tc.RawInput != nil {
					printInput(tc.RawInput)
					st.inputShown = true
				}
				printContent(tc.Content)
			}

		case u.ToolCallUpdate != nil:
			tc := u.ToolCallUpdate
			id := string(tc.ToolCallId)

			st := toolCalls[id]
			if st == nil {
				st = &toolState{}
				toolCalls[id] = st
			}
			if tc.Title != nil {
				st.title = *tc.Title
			}
			if tc.Kind != nil {
				st.kind = string(*tc.Kind)
			}

			newStatus := acp.ToolCallStatus("")
			if tc.Status != nil {
				newStatus = *tc.Status
			}

			// Only print header for actual status changes.
			if newStatus != "" && newStatus != st.status {
				spin.stop()
				ensureNewline()
				st.status = newStatus
				printToolHeader(st.title, string(newStatus), st.kind)
			}

			printLocations(tc.Locations)

			// Show input once (on first update that provides it).
			if tc.RawInput != nil && !st.inputShown {
				printInput(tc.RawInput)
				st.inputShown = true
			}

			printOutput(tc.RawOutput)
			printContent(tc.Content)

			if newStatus == acp.ToolCallStatusCompleted || newStatus == acp.ToolCallStatusFailed {
				delete(toolCalls, id)
			}
		}
	}

	statusCh := make(chan v2.SessionStatus, 8)

	fmt.Fprintf(os.Stderr, "launching %s session (mode=%s, cwd=%s)...\n", *agent, *mode, absCwd)

	sess, err := drv.Launch(ctx, v2.LaunchOpts{
		Prompt:       prompt,
		SystemPrompt: *system,
		Model:        *model,
		Cwd:          absCwd,
		SessionMode:  *mode,
		MCPServers:   []acp.McpServer{},
		StatusCh:     statusCh,
	}, onEvent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: launch failed: %v\n", err)
		os.Exit(1)
	}

	spin.start()

	// Auto-approve permissions in the background.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case id := <-permCh:
				_ = sess.RespondToPermission(ctx, id, true, "")
			}
		}
	}()

	// Wait for the initial prompt to finish (session goes idle).
	waitForIdle(ctx, sess, statusCh, spin, ensureNewline)
	spin.stop()

	if ctx.Err() != nil {
		_ = sess.Stop(context.Background())
		return
	}

	fmt.Println() // newline after streamed output

	// Interactive read loop.
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, "\n> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" {
			break
		}

		spin.start()
		if cmdName, _, ok := parseSlashCommand(input); ok {
			fmt.Fprintf(os.Stderr, "\033[2m[slash command sent via session/prompt: /%s]\033[0m\n", cmdName)
			if _, known := availableCommands[cmdName]; !known {
				fmt.Fprintf(os.Stderr, "\033[2m[slash command not in latest availableCommands: /%s]\033[0m\n", cmdName)
			}
		}
		before := atomic.LoadUint64(&updateCount)
		resp, err := sess.Prompt(ctx, []acp.ContentBlock{acp.TextBlock(input)})
		after := atomic.LoadUint64(&updateCount)
		spin.stop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: prompt failed: %v\n", err)
			break
		}
		if after == before {
			fmt.Fprintf(os.Stderr, "\033[2m[prompt done: stopReason=%s, no session updates emitted]\033[0m\n", resp.StopReason)
		}
		fmt.Println()
	}

	fmt.Fprintln(os.Stderr, "\nstopping session...")
	_ = sess.Stop(context.Background())
}

// waitForIdle waits for the session to reach idle, stopped, or errored status
// using push-based notifications via statusCh.
func waitForIdle(ctx context.Context, sess v2.Session, statusCh <-chan v2.SessionStatus, spin *spinner, ensureNewline func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case status := <-statusCh:
			spin.stop()
			ensureNewline()
			info := sess.Info()
			fmt.Fprintf(os.Stderr, "\033[2m[session: %s", status)
			if info.AgentSessionID != "" {
				fmt.Fprintf(os.Stderr, " agent=%s", info.AgentSessionID)
			}
			fmt.Fprint(os.Stderr, "]\033[0m\n")
			switch status {
			case v2.SessionStatusRunning, v2.SessionStatusStarting:
				spin.start()
			case v2.SessionStatusIdle, v2.SessionStatusStopped, v2.SessionStatusErrored:
				return
			}
		}
	}
}

func statusLabel(s string) string {
	switch s {
	case "pending":
		return "⏳"
	case "in_progress":
		return "⚙️"
	case "completed":
		return "✓"
	case "failed":
		return "✗"
	default:
		return "…"
	}
}

func printToolHeader(title, status, kind string) {
	kindStr := ""
	if kind != "" {
		kindStr = fmt.Sprintf(" (%s)", kind)
	}
	fmt.Fprintf(os.Stderr, "\033[36m[tool: %s %s%s]\033[0m\n",
		title, statusLabel(status), kindStr)
}

func printLocations(locs []acp.ToolCallLocation) {
	for _, loc := range locs {
		if loc.Line != nil {
			fmt.Fprintf(os.Stderr, "\033[2m  📍 %s:%d\033[0m\n", loc.Path, *loc.Line)
		} else {
			fmt.Fprintf(os.Stderr, "\033[2m  📍 %s\033[0m\n", loc.Path)
		}
	}
}

func printInput(raw any) {
	if raw == nil {
		return
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for k, v := range m {
		s := fmt.Sprintf("%v", v)
		if len(s) > 200 {
			s = s[:197] + "..."
		}
		// Indent multiline values.
		if strings.Contains(s, "\n") {
			lines := strings.Split(s, "\n")
			fmt.Fprintf(os.Stderr, "\033[2m  %s:\033[0m\n", k)
			for _, line := range lines {
				fmt.Fprintf(os.Stderr, "\033[2m    %s\033[0m\n", line)
			}
		} else {
			fmt.Fprintf(os.Stderr, "\033[2m  %s: %s\033[0m\n", k, s)
		}
	}
}

func printOutput(raw any) {
	if raw == nil {
		return
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return
	}
	s := string(b)
	if s == "null" || s == "{}" || s == "\"\"" {
		return
	}
	if len(s) > 500 {
		s = s[:497] + "..."
	}
	fmt.Fprintf(os.Stderr, "\033[2m  → %s\033[0m\n", s)
}

func printContent(content []acp.ToolCallContent) {
	for _, c := range content {
		if c.Diff != nil {
			fmt.Fprintf(os.Stderr, "\033[33m  diff: %s\033[0m\n", c.Diff.Path)
			if c.Diff.OldText != nil {
				for _, line := range strings.Split(*c.Diff.OldText, "\n") {
					fmt.Fprintf(os.Stderr, "\033[31m  - %s\033[0m\n", line)
				}
			}
			for _, line := range strings.Split(c.Diff.NewText, "\n") {
				fmt.Fprintf(os.Stderr, "\033[32m  + %s\033[0m\n", line)
			}
		}
		if c.Content != nil && c.Content.Content.Text != nil {
			text := c.Content.Content.Text.Text
			if len(text) > 500 {
				text = text[:497] + "..."
			}
			fmt.Fprintf(os.Stderr, "\033[2m  content: %s\033[0m\n", text)
		}
	}
}

func agentConfig(name string) (v2.AgentConfig, error) {
	switch name {
	case "claude-code":
		cfg := v2.ClaudeCodeConfig
		cfg.AdapterFactory = claudeacp.NewAdapter
		return cfg, nil
	case "codex":
		cfg := v2.CodexConfig
		cfg.AdapterFactory = codexacp.NewAdapter
		return cfg, nil
	case "opencode":
		return v2.OpenCodeConfig, nil
	case "gemini":
		return v2.GeminiConfig, nil
	default:
		return v2.AgentConfig{}, fmt.Errorf("unknown agent: %s (use claude-code, codex, opencode, gemini)", name)
	}
}

func stdinHasData() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

func parseSlashCommand(input string) (name, arg string, ok bool) {
	if !strings.HasPrefix(input, "/") {
		return "", "", false
	}
	trimmed := strings.TrimSpace(strings.TrimPrefix(input, "/"))
	if trimmed == "" {
		return "", "", false
	}
	parts := strings.SplitN(trimmed, " ", 2)
	name = strings.TrimSpace(parts[0])
	if name == "" {
		return "", "", false
	}
	if len(parts) == 2 {
		arg = strings.TrimSpace(parts[1])
	}
	return name, arg, true
}

func rawFallback(update acp.SessionUpdate, marshalErr error) map[string]any {
	out := map[string]any{
		"marshalError": marshalErr.Error(),
	}
	switch {
	case update.UserMessageChunk != nil:
		out["sessionUpdate"] = "user_message_chunk"
	case update.AgentMessageChunk != nil:
		out["sessionUpdate"] = "agent_message_chunk"
		if update.AgentMessageChunk.Content.Text != nil {
			out["text"] = update.AgentMessageChunk.Content.Text.Text
		}
	case update.AgentThoughtChunk != nil:
		out["sessionUpdate"] = "agent_thought_chunk"
		if update.AgentThoughtChunk.Content.Text != nil {
			out["text"] = update.AgentThoughtChunk.Content.Text.Text
		}
	case update.ToolCall != nil:
		out["sessionUpdate"] = "tool_call"
		out["toolCallId"] = update.ToolCall.ToolCallId
		out["title"] = update.ToolCall.Title
		out["status"] = update.ToolCall.Status
		out["kind"] = update.ToolCall.Kind
	case update.ToolCallUpdate != nil:
		out["sessionUpdate"] = "tool_call_update"
		out["toolCallId"] = update.ToolCallUpdate.ToolCallId
		if update.ToolCallUpdate.Title != nil {
			out["title"] = *update.ToolCallUpdate.Title
		}
		if update.ToolCallUpdate.Status != nil {
			out["status"] = *update.ToolCallUpdate.Status
		}
		if update.ToolCallUpdate.Kind != nil {
			out["kind"] = *update.ToolCallUpdate.Kind
		}
	case update.Plan != nil:
		out["sessionUpdate"] = "plan"
		out["entries"] = len(update.Plan.Entries)
	case update.CurrentModeUpdate != nil:
		out["sessionUpdate"] = "current_mode_update"
		out["currentModeId"] = update.CurrentModeUpdate.CurrentModeId
	case update.AvailableCommandsUpdate != nil:
		out["sessionUpdate"] = "available_commands_update"
		out["availableCommands"] = update.AvailableCommandsUpdate.AvailableCommands
	default:
		out["sessionUpdate"] = "unknown"
	}
	return out
}

func tryMarshalChunkRaw(update acp.SessionUpdate) ([]byte, bool) {
	switch {
	case update.UserMessageChunk != nil:
		out := map[string]any{
			"sessionUpdate": "user_message_chunk",
		}
		if update.UserMessageChunk.Content.Text != nil {
			out["content"] = map[string]any{
				"type": "text",
				"text": update.UserMessageChunk.Content.Text.Text,
			}
		}
		b, err := json.Marshal(out)
		return b, err == nil
	case update.AgentMessageChunk != nil:
		out := map[string]any{
			"sessionUpdate": "agent_message_chunk",
		}
		if update.AgentMessageChunk.Content.Text != nil {
			out["content"] = map[string]any{
				"type": "text",
				"text": update.AgentMessageChunk.Content.Text.Text,
			}
		}
		b, err := json.Marshal(out)
		return b, err == nil
	case update.AgentThoughtChunk != nil:
		out := map[string]any{
			"sessionUpdate": "agent_thought_chunk",
		}
		if update.AgentThoughtChunk.Content.Text != nil {
			out["content"] = map[string]any{
				"type": "text",
				"text": update.AgentThoughtChunk.Content.Text.Text,
			}
		}
		b, err := json.Marshal(out)
		return b, err == nil
	default:
		return nil, false
	}
}
