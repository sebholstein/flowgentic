package workload

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// AgentRunManager manages agent drivers and sessions.
type AgentRunManager struct {
	log       *slog.Logger
	drivers   map[string]driver.Driver
	ctlURL    string
	ctlSecret string
	mu        sync.RWMutex
	sessions  map[string]sessionEntry
	cwdMu     sync.Mutex
	cwdLocks  map[string]*sync.Mutex
}

type sessionEntry struct {
	session driver.Session
	driver  driver.Driver
}

// NewAgentRunManager creates a new AgentRunManager with the given drivers.
func NewAgentRunManager(log *slog.Logger, ctlURL, ctlSecret string, drivers ...driver.Driver) *AgentRunManager {
	dm := make(map[string]driver.Driver, len(drivers))
	for _, d := range drivers {
		dm[d.Agent()] = d
	}
	return &AgentRunManager{
		log:       log,
		drivers:   dm,
		ctlURL:    ctlURL,
		ctlSecret: ctlSecret,
		sessions:  make(map[string]sessionEntry),
		cwdLocks:  make(map[string]*sync.Mutex),
	}
}

// Launch starts a new session with the specified agent driver.
// agentRunID is the control-plane-provided key used for session tracking,
// event routing, and hook callbacks. opts.SessionID is only used by the
// driver to resume an existing agent session.
func (m *AgentRunManager) Launch(_ context.Context, agentRunID, agentID string, opts driver.LaunchOpts, onEvent driver.EventCallback) (driver.Session, error) {
	ctx := context.Background()
	d, ok := m.drivers[agentID]
	if !ok {
		return nil, fmt.Errorf("unknown agent driver: %s", agentID)
	}

	caps := d.Capabilities()
	if opts.SessionID != "" && !caps.Has(driver.CapSessionResume) {
		return nil, fmt.Errorf("agent %s does not support session resume", agentID)
	}
	if opts.Model != "" && !caps.Has(driver.CapCustomModel) {
		return nil, fmt.Errorf("agent %s does not support custom model selection", agentID)
	}
	if opts.SystemPrompt != "" && !caps.Has(driver.CapSystemPrompt) {
		return nil, fmt.Errorf("agent %s does not support system prompts", agentID)
	}
	if opts.Yolo && !caps.Has(driver.CapYolo) {
		return nil, fmt.Errorf("agent %s does not support yolo mode", agentID)
	}

	// Inject CTL env vars so agents can reach the private listener.
	if opts.EnvVars == nil {
		opts.EnvVars = make(map[string]string)
	}
	opts.EnvVars["AGENTCTL_WORKER_URL"] = m.ctlURL
	opts.EnvVars["AGENTCTL_WROKER_SECRET"] = m.ctlSecret
	opts.EnvVars["AGENTCTL_AGENT_RUN_ID"] = agentRunID

	wrappedOnEvent := func(e driver.Event) {
		e.Agent = agentID
		m.log.Info("event received",
			"type", e.Type,
			"session_id", e.SessionID,
			"agent_id", e.Agent,
			"error", e.Error,
			"text", e.Text,
			"stop_reason", e.StopReason,
			"tool_name", e.ToolName,
			"tool_id", e.ToolID,
			"tool_input", string(e.ToolInput),
		)
		if onEvent != nil {
			onEvent(e)
		}

		// Auto-remove session when it stops.
		if e.Type == driver.EventTypeTurnComplete {
			m.removeSession(agentRunID)
		}
	}

	// If the driver needs post-launch session ID resolution, serialize
	// launches per working directory to avoid races during discovery.
	resolver, needsResolve := d.(driver.SessionResolver)
	if needsResolve {
		cwdLock := m.getCwdLock(opts.Cwd)
		cwdLock.Lock()
		defer cwdLock.Unlock()
	}

	sess, err := d.Launch(ctx, opts, wrappedOnEvent)
	if err != nil {
		return nil, fmt.Errorf("launch %s: %w", agentID, err)
	}

	// Only call ResolveSessionID if Launch didn't already set the ID
	// (e.g. headless Codex sets it synchronously from the stream).
	if needsResolve && sess.Info().AgentSessionID == "" {
		agentSessID, resolveErr := resolver.ResolveSessionID(ctx, opts.Cwd)
		if resolveErr != nil {
			m.log.Warn("failed to resolve agent session ID", "agent", agentID, "error", resolveErr)
		} else if agentSessID != "" {
			if setter, ok := sess.(driver.AgentSessionIDSetter); ok {
				setter.SetAgentSessionID(agentSessID)
			}
		}
	}

	m.mu.Lock()
	m.sessions[agentRunID] = sessionEntry{session: sess, driver: d}
	m.mu.Unlock()

	m.log.Info("session launched", "agent", agentID, "agent_run_id", agentRunID, "mode", opts.Mode, "agent_session_id", sess.Info().AgentSessionID)
	return sess, nil
}

func (m *AgentRunManager) getCwdLock(cwd string) *sync.Mutex {
	m.cwdMu.Lock()
	defer m.cwdMu.Unlock()
	lock, ok := m.cwdLocks[cwd]
	if !ok {
		lock = &sync.Mutex{}
		m.cwdLocks[cwd] = lock
	}
	return lock
}

// GetSession returns a session by ID.
func (m *AgentRunManager) GetSession(id string) (driver.Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.sessions[id]
	if !ok {
		return nil, false
	}
	return e.session, true
}

// SessionListEntry pairs a workload ID (agent_run_id) with its session info.
type SessionListEntry struct {
	AgentRunID string
	Info       driver.SessionInfo
}

// ListSessions returns info for all active sessions, keyed by agent run ID.
func (m *AgentRunManager) ListSessions() []SessionListEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := make([]SessionListEntry, 0, len(m.sessions))
	for id, e := range m.sessions {
		entries = append(entries, SessionListEntry{
			AgentRunID: id,
			Info:       e.session.Info(),
		})
	}
	return entries
}

// StopSession stops the session with the given ID.
func (m *AgentRunManager) StopSession(ctx context.Context, id string) error {
	m.mu.RLock()
	e, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	if err := e.session.Stop(ctx); err != nil {
		return err
	}
	m.removeSession(id)
	return nil
}

// ListDrivers returns capabilities for all registered drivers.
func (m *AgentRunManager) ListDrivers() []driver.Capabilities {
	caps := make([]driver.Capabilities, 0, len(m.drivers))
	for _, d := range m.drivers {
		caps = append(caps, d.Capabilities())
	}
	return caps
}

// HandleHookEvent routes an incoming hook event to the appropriate session's driver.
func (m *AgentRunManager) HandleHookEvent(ctx context.Context, event driver.HookEvent) error {
	entry := m.getSessionEntry(event.SessionID)
	if entry == nil {
		return fmt.Errorf("session not found: %s", event.SessionID)
	}
	return entry.driver.HandleHookEvent(ctx, event.SessionID, event)
}

func (m *AgentRunManager) getSessionEntry(id string) *sessionEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.sessions[id]
	if !ok {
		return nil
	}
	return &e
}

// HandleStatusReport processes a status update from an agent.
func (m *AgentRunManager) HandleStatusReport(_ context.Context, sessionID, agent, status string) error {
	entry := m.getSessionEntry(sessionID)
	if entry == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	m.log.Info("status report", "session_id", sessionID, "agent", agent, "status", status)
	return nil
}

// HandlePlanSubmission processes a plan submission from an agent.
func (m *AgentRunManager) HandlePlanSubmission(_ context.Context, sessionID, agent string, plan []byte) error {
	entry := m.getSessionEntry(sessionID)
	if entry == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	m.log.Info("plan submitted", "session_id", sessionID, "agent", agent, "plan_bytes", len(plan))
	return nil
}

func (m *AgentRunManager) removeSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}
