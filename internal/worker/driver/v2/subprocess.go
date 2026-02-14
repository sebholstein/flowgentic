package v2

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/google/uuid"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// acpDriver implements Driver using ACP connections.
type acpDriver struct {
	log    *slog.Logger
	config AgentConfig
	caps   driver.Capabilities
}

// NewDriver creates a V2 driver from an AgentConfig.
func NewDriver(log *slog.Logger, config AgentConfig) Driver {
	return &acpDriver{
		log:    log.With("driver", config.AgentID),
		config: config,
		caps: driver.Capabilities{
			Agent:     config.AgentID,
			Supported: config.Capabilities,
		},
	}
}

func (d *acpDriver) Agent() string                     { return d.config.AgentID }
func (d *acpDriver) Capabilities() driver.Capabilities { return d.caps }

func (d *acpDriver) DiscoverModels(ctx context.Context, cwd string) (ModelInventory, error) {
	client := newFlowgenticClient(nil, nil, "")

	var (
		conn *acp.ClientSideConnection
		cmd  *exec.Cmd
		err  error
	)

	if d.config.AdapterFactory != nil {
		conn, cmd, err = d.launchInProcess(ctx, client, LaunchOpts{Cwd: cwd})
	} else if d.config.Command != "" {
		conn, cmd, err = d.launchSubprocess(ctx, client, LaunchOpts{Cwd: cwd})
	} else {
		return ModelInventory{}, fmt.Errorf("agent config has neither AdapterFactory nor Command")
	}
	if err != nil {
		return ModelInventory{}, err
	}

	if cmd != nil {
		defer func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}()
	}

	_, err = conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersion(acp.ProtocolVersionNumber),
		ClientInfo: &acp.Implementation{
			Name:    "flowgentic",
			Version: "1.0.0",
		},
	})
	if err != nil {
		return ModelInventory{}, fmt.Errorf("ACP initialize failed: %w", err)
	}

	resp, err := conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        cwd,
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		return ModelInventory{}, fmt.Errorf("ACP new session failed: %w", err)
	}
	if resp.Models == nil {
		return ModelInventory{}, fmt.Errorf("ACP agent %s returned no model metadata", d.config.AgentID)
	}

	models := make([]string, 0, len(resp.Models.AvailableModels))
	for _, m := range resp.Models.AvailableModels {
		if m.ModelId == "" {
			continue
		}
		models = append(models, string(m.ModelId))
	}
	defaultModel := string(resp.Models.CurrentModelId)
	if len(models) == 0 || defaultModel == "" {
		return ModelInventory{}, fmt.Errorf("ACP agent %s returned incomplete model metadata", d.config.AgentID)
	}

	return ModelInventory{
		Models:       models,
		DefaultModel: defaultModel,
	}, nil
}

func (d *acpDriver) Launch(ctx context.Context, opts LaunchOpts, onEvent EventCallback) (Session, error) {
	sessionID := opts.ResumeSessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	info := SessionInfo{
		ID:        sessionID,
		AgentID:   d.config.AgentID,
		Status:    SessionStatusStarting,
		Cwd:       opts.Cwd,
		StartedAt: time.Now(),
	}

	client := newFlowgenticClient(onEvent, opts.Handlers, opts.SessionMode)

	launchCtx, cancel := context.WithCancel(ctx)

	sess := &acpSession{
		info:     info,
		client:   client,
		cancel:   cancel,
		done:     make(chan struct{}),
		statusCh: opts.StatusCh,
		promptCh: make(chan promptRequest),
		cancelCh: make(chan struct{}, 1),
	}

	var (
		conn *acp.ClientSideConnection
		cmd  *exec.Cmd
	)

	if d.config.AdapterFactory != nil {
		// In-process adapter: use io.Pipe pairs.
		var err error
		conn, cmd, err = d.launchInProcess(launchCtx, client, opts)
		if err != nil {
			cancel()
			return nil, err
		}
		_ = cmd // nil for in-process
	} else if d.config.Command != "" {
		// Subprocess: spawn external ACP agent.
		var err error
		conn, cmd, err = d.launchSubprocess(launchCtx, client, opts)
		if err != nil {
			cancel()
			return nil, err
		}
	} else {
		cancel()
		return nil, fmt.Errorf("agent config has neither AdapterFactory nor Command")
	}

	sess.conn = conn

	// Run the ACP Initialize → NewSession → Prompt flow in a goroutine.
	go d.runSession(launchCtx, sess, conn, cmd, opts)

	return sess, nil
}

// ConnectionSetter is implemented by in-process adapters that need a reference
// to the agent-side connection for sending notifications.
type ConnectionSetter interface {
	SetConnection(conn *acp.AgentSideConnection)
}

func (d *acpDriver) launchInProcess(_ context.Context, client *flowgenticClient, opts LaunchOpts) (*acp.ClientSideConnection, *exec.Cmd, error) {
	agent := d.config.AdapterFactory(d.log)

	// Two pipe pairs: client writes to agent's stdin, agent writes to client's stdin.
	clientToAgentR, clientToAgentW := io.Pipe()
	agentToClientR, agentToClientW := io.Pipe()

	// Client side: writes to clientToAgentW (agent's stdin), reads from agentToClientR (agent's stdout).
	conn := acp.NewClientSideConnection(client, clientToAgentW, agentToClientR)
	conn.SetLogger(d.log.With("side", "client"))

	// Agent side: writes to agentToClientW (client's stdin), reads from clientToAgentR (client's stdout).
	agentConn := acp.NewAgentSideConnection(agent, agentToClientW, clientToAgentR)
	agentConn.SetLogger(d.log.With("side", "agent"))

	// Give the adapter a reference to its connection for sending notifications.
	if setter, ok := agent.(ConnectionSetter); ok {
		setter.SetConnection(agentConn)
	}

	_ = opts // env vars not applicable for in-process

	return conn, nil, nil
}

func (d *acpDriver) launchSubprocess(ctx context.Context, client *flowgenticClient, opts LaunchOpts) (*acp.ClientSideConnection, *exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, d.config.Command, d.config.Args...)
	cmd.Env = driver.BuildEnv(opts.EnvVars)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start %s: %w", d.config.Command, err)
	}

	conn := acp.NewClientSideConnection(client, stdin, stdout)
	conn.SetLogger(d.log)

	return conn, cmd, nil
}

func (d *acpDriver) runSession(ctx context.Context, sess *acpSession, conn *acp.ClientSideConnection, cmd *exec.Cmd, opts LaunchOpts) {
	defer func() {
		sess.client.closePendingPermissions()
		sess.setStatus(SessionStatusStopped)
		// Close the status channel so consumers (e.g. forwardStatusEvents) exit.
		sess.closeStatusCh()
		close(sess.done)
		// Wait for subprocess to exit if applicable.
		if cmd != nil {
			_ = cmd.Wait()
		}
	}()

	// Step 1: Initialize
	initResp, err := conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersion(acp.ProtocolVersionNumber),
		ClientInfo: &acp.Implementation{
			Name:    "flowgentic",
			Version: "1.0.0",
		},
	})
	if err != nil {
		d.log.Error("ACP initialize failed", "error", err)
		sess.setStatus(SessionStatusErrored)
		return
	}
	d.log.Info("ACP initialized", "agent_info", initResp.AgentInfo, "protocol_version", initResp.ProtocolVersion)

	// Step 2: NewSession (or LoadSession if resuming)
	meta := d.buildMeta(opts)
	mcpServers := sessionMCPServers(opts)
	var sessionID acp.SessionId

	if opts.ResumeSessionID != "" {
		// Resume an existing session.
		_, loadErr := conn.LoadSession(ctx, acp.LoadSessionRequest{
			SessionId:  acp.SessionId(opts.ResumeSessionID),
			Cwd:        opts.Cwd,
			McpServers: mcpServers,
		})
		if loadErr != nil {
			d.log.Error("ACP load session failed", "error", loadErr)
			sess.setStatus(SessionStatusErrored)
			return
		}
		sessionID = acp.SessionId(opts.ResumeSessionID)
		d.log.Info("ACP session loaded", "agent_session_id", sessionID)
	} else {
		newSessResp, newErr := conn.NewSession(ctx, acp.NewSessionRequest{
			Cwd:        opts.Cwd,
			Meta:       meta,
			McpServers: mcpServers,
		})
		if newErr != nil {
			d.log.Error("ACP new session failed", "error", newErr)
			sess.setStatus(SessionStatusErrored)
			return
		}
		sessionID = newSessResp.SessionId
		d.log.Info("ACP session created", "agent_session_id", sessionID)

		sess.mu.Lock()
		if newSessResp.Models != nil {
			models := make([]string, 0, len(newSessResp.Models.AvailableModels))
			for _, m := range newSessResp.Models.AvailableModels {
				if m.ModelId == "" {
					continue
				}
				models = append(models, string(m.ModelId))
			}
			sess.info.Models = models
			if newSessResp.Models.CurrentModelId != "" {
				sess.info.CurrentModel = string(newSessResp.Models.CurrentModelId)
			}
		}
		if newSessResp.Modes != nil {
			modes := make([]string, 0, len(newSessResp.Modes.AvailableModes))
			for _, m := range newSessResp.Modes.AvailableModes {
				if m.Id == "" {
					continue
				}
				modes = append(modes, string(m.Id))
			}
			sess.info.Modes = modes
		}
		sess.mu.Unlock()
	}

	sess.mu.Lock()
	sess.info.AgentSessionID = string(sessionID)
	sess.mu.Unlock()
	sess.setStatus(SessionStatusRunning)

	// Step 3: Initial prompt (optional).
	if strings.TrimSpace(opts.Prompt) != "" {
		// For subprocess agents, _meta.systemPrompt is non-standard and may be
		// ignored (e.g. OpenCode). Prepend it to the prompt text so the agent
		// always sees it. In-process adapters (Claude Code) handle systemPrompt
		// via their own NewSession/Prompt logic and skip this path.
		var blocks []acp.ContentBlock
		if cmd != nil && opts.SystemPrompt != "" {
			blocks = append(blocks, acp.TextBlock(opts.SystemPrompt+"\n\n---\n\n"))
		}
		blocks = append(blocks, acp.TextBlock(opts.Prompt))

		promptResp, promptErr := d.doPrompt(ctx, conn, sessionID, blocks)
		if promptErr != nil {
			if ctx.Err() != nil {
				d.log.Info("ACP session cancelled")
				return
			}
			d.log.Error("ACP prompt failed", "error", promptErr)
			sess.setStatus(SessionStatusErrored)
			return
		}
		d.log.Info("ACP prompt completed", "stop_reason", promptResp.StopReason)
	}

	// Step 4: Enter idle loop — wait for follow-up prompts or cancellation.
	sess.setStatus(SessionStatusIdle)

	for {
		select {
		case req := <-sess.promptCh:
			sess.setStatus(SessionStatusRunning)
			resp, pErr := d.doPrompt(ctx, conn, sessionID, req.blocks)
			req.resultCh <- promptResult{resp: resp, err: pErr}
			if pErr != nil && ctx.Err() != nil {
				return
			}
			sess.setStatus(SessionStatusIdle)

		case <-sess.cancelCh:
			// Drain any pending cancel signals.
			for len(sess.cancelCh) > 0 {
				<-sess.cancelCh
			}
			d.log.Info("ACP session cancel requested")

		case <-ctx.Done():
			return
		}
	}
}

// doPrompt sends a single prompt turn to the ACP connection.
func (d *acpDriver) doPrompt(ctx context.Context, conn *acp.ClientSideConnection, sessionID acp.SessionId, blocks []acp.ContentBlock) (*acp.PromptResponse, error) {
	resp, err := conn.Prompt(ctx, acp.PromptRequest{
		SessionId: sessionID,
		Prompt:    blocks,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (d *acpDriver) buildMeta(opts LaunchOpts) map[string]any {
	if d.config.MetaBuilder != nil {
		return d.config.MetaBuilder(opts)
	}
	return defaultMetaBuilder(opts)
}

func sessionMCPServers(opts LaunchOpts) []acp.McpServer {
	servers := make([]acp.McpServer, len(opts.MCPServers))
	copy(servers, opts.MCPServers)
	if !shouldInjectDefaultFlowgenticMCP(opts) {
		return servers
	}
	if flowgentic, ok := defaultFlowgenticMCPServer(opts.EnvVars); ok && !hasStdioMCPServerNamed(servers, flowgentic.Stdio.Name) {
		// Keep a stable trace of the exact binary used for the session-scoped MCP server.
		slog.Default().Info("injecting flowgentic MCP server", "command", flowgentic.Stdio.Command, "args", flowgentic.Stdio.Args)
		servers = append(servers, flowgentic)
	}
	return servers
}

func shouldInjectDefaultFlowgenticMCP(opts LaunchOpts) bool {
	if strings.EqualFold(strings.TrimSpace(opts.EnvVars["FLOWGENTIC_ENABLE_DEFAULT_MCP"]), "1") {
		return true
	}
	return strings.Contains(opts.SystemPrompt, "## Flowgentic MCP")
}

func defaultFlowgenticMCPServer(envVars map[string]string) (acp.McpServer, bool) {
	if strings.TrimSpace(envVars["AGENTCTL_WORKER_URL"]) == "" || strings.TrimSpace(envVars["AGENTCTL_AGENT_RUN_ID"]) == "" {
		return acp.McpServer{}, false
	}

	env := []acp.EnvVariable{
		{Name: "AGENTCTL_WORKER_URL", Value: envVars["AGENTCTL_WORKER_URL"]},
		{Name: "AGENTCTL_WORKER_SECRET", Value: envVars["AGENTCTL_WORKER_SECRET"]},
		{Name: "AGENTCTL_AGENT_RUN_ID", Value: envVars["AGENTCTL_AGENT_RUN_ID"]},
		{Name: "AGENTCTL_AGENT", Value: envVars["AGENTCTL_AGENT"]},
	}

	command, commandArgs := resolveAgentctlInvocation(envVars)
	return acp.McpServer{
		Stdio: &acp.McpServerStdio{
			Name:    "flowgentic",
			Command: command,
			Args:    commandArgs,
			Env:     env,
		},
	}, true
}

func resolveAgentctlInvocation(envVars map[string]string) (string, []string) {
	candidates := make([]string, 0, 4)
	if bin := strings.TrimSpace(envVars["AGENTCTL_BIN"]); bin != "" {
		candidates = append(candidates, bin)
	}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		candidate := filepath.Join(cwd, "bin", "agentctl")
		if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
			candidates = append(candidates, candidate)
		}
	}

	if workerExe, err := os.Executable(); err == nil && workerExe != "" {
		candidate := filepath.Join(filepath.Dir(workerExe), "agentctl")
		if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
			candidates = append(candidates, candidate)
		}
	}

	if path, err := exec.LookPath("agentctl"); err == nil && path != "" {
		candidates = append(candidates, path)
	}

	for _, candidate := range candidates {
		if commandSupportsMCPServe(candidate) {
			return candidate, nil
		}
	}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		sourceCmdDir := filepath.Join(cwd, "cmd", "agentctl")
		if _, statErr := os.Stat(sourceCmdDir); statErr == nil {
			slog.Default().Warn("flowgentic MCP using PATH agentctl; build/install `agentctl` for best compatibility")
		}
	}

	return "agentctl", nil
}

func commandSupportsMCPServe(command string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	// agentctl runs as an MCP stdio server directly — verify the binary exists
	// and is executable by invoking it with --help (which will fail quickly for
	// non-agentctl binaries).
	err := exec.CommandContext(ctx, command, "--help").Run()
	return err == nil
}

func hasStdioMCPServerNamed(servers []acp.McpServer, name string) bool {
	for _, server := range servers {
		if server.Stdio != nil && server.Stdio.Name == name {
			return true
		}
	}
	return false
}
