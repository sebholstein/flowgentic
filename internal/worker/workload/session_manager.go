package workload

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	acp "github.com/coder/acp-go-sdk"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
)

// StateEventType describes what kind of state change occurred.
type StateEventType int

const (
	// StateEventUpdate indicates a session was added or changed.
	StateEventUpdate StateEventType = iota
	// StateEventRemoved indicates a session was removed.
	StateEventRemoved
)

// StateEvent carries information about a state change.
type StateEvent struct {
	Type      StateEventType
	SessionID string
	// Snapshot is set for Update events, nil for Removed.
	Snapshot *SessionSnapshot
}

// SessionEventUpdate carries a raw session event for subscribers.
type SessionEventUpdate struct {
	SessionID string
	Event     *workerv1.SessionEvent
}

// SessionManager manages agent drivers and sessions.
type SessionManager struct {
	log         *slog.Logger
	drivers     map[string]v2.Driver
	ctlURL      string
	ctlSecret   string
	mu          sync.RWMutex
	sessions    map[string]*sessionEntry
	subscribers map[chan StateEvent]struct{}

	eventQueue       *EventQueue
	eventSubscribers map[chan SessionEventUpdate]struct{}
}

type sessionEntry struct {
	session v2.Session
	driver  v2.Driver
	topic   string
	nextSeq atomic.Int64
}

// NewSessionManager creates a new SessionManager with the given drivers.
func NewSessionManager(log *slog.Logger, ctlURL, ctlSecret string, drivers ...v2.Driver) *SessionManager {
	dm := make(map[string]v2.Driver, len(drivers))
	for _, d := range drivers {
		dm[d.Agent()] = d
	}

	return &SessionManager{
		log:              log,
		drivers:          dm,
		ctlURL:           ctlURL,
		ctlSecret:        ctlSecret,
		sessions:         make(map[string]*sessionEntry),
		subscribers:      make(map[chan StateEvent]struct{}),
		eventQueue:       NewEventQueue(),
		eventSubscribers: make(map[chan SessionEventUpdate]struct{}),
	}
}

// Launch starts a new session with the specified agent driver.
func (m *SessionManager) Launch(_ context.Context, sessionID, agentID string, opts v2.LaunchOpts, onEvent v2.EventCallback) (v2.Session, error) {
	ctx := context.Background()
	d, ok := m.drivers[agentID]
	if !ok {
		return nil, fmt.Errorf("unknown agent driver: %s", agentID)
	}

	caps := d.Capabilities()
	if opts.ResumeSessionID != "" && !caps.Has(driver.CapSessionResume) {
		return nil, fmt.Errorf("agent %s does not support session resume", agentID)
	}
	if opts.Model != "" && !caps.Has(driver.CapCustomModel) {
		return nil, fmt.Errorf("agent %s does not support custom model selection", agentID)
	}
	if opts.SystemPrompt != "" && !caps.Has(driver.CapSystemPrompt) {
		return nil, fmt.Errorf("agent %s does not support system prompts", agentID)
	}
	// Inject CTL env vars so agents can reach the private listener.
	if opts.EnvVars == nil {
		opts.EnvVars = make(map[string]string)
	}
	opts.EnvVars["AGENTCTL_WORKER_URL"] = m.ctlURL
	opts.EnvVars["AGENTCTL_WORKER_SECRET"] = m.ctlSecret
	opts.EnvVars["AGENTCTL_AGENT_RUN_ID"] = sessionID
	opts.EnvVars["AGENTCTL_AGENT"] = agentID

	entry := &sessionEntry{}

	wrappedOnEvent := func(n acp.SessionNotification) {
		logACPEvent(m.log, agentID, n)
		m.emitSessionEvent(sessionID, entry, n)
		if onEvent != nil {
			onEvent(n)
		}
	}

	// Wire up status channel so transitions (e.g. running→idle) are emitted
	// as SessionEvent_StatusChange. This triggers the assembler to flush any
	// buffered text/thought chunks that haven't been persisted yet.
	statusCh := make(chan v2.SessionStatus, 4)
	opts.StatusCh = statusCh
	go m.forwardStatusEvents(sessionID, entry, statusCh)

	sess, err := d.Launch(ctx, opts, wrappedOnEvent)
	if err != nil {
		return nil, fmt.Errorf("launch %s: %w", agentID, err)
	}

	entry.session = sess
	entry.driver = d

	m.mu.Lock()
	m.sessions[sessionID] = entry
	m.mu.Unlock()

	snap := SessionSnapshot{SessionID: sessionID, Info: sess.Info()}
	m.notifySubscribers(StateEvent{Type: StateEventUpdate, SessionID: sessionID, Snapshot: &snap})

	m.log.Info("session launched", "agent", agentID, "session_id", sessionID, "agent_session_id", sess.Info().AgentSessionID)
	return sess, nil
}

// GetSession returns a session by ID.
func (m *SessionManager) GetSession(id string) (v2.Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.sessions[id]
	if !ok {
		return nil, false
	}
	return e.session, true
}

// SessionListEntry pairs a session ID with its session info.
type SessionListEntry struct {
	SessionID string
	Info      v2.SessionInfo
}

// ListSessions returns info for all active sessions, keyed by session ID.
func (m *SessionManager) ListSessions() []SessionListEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := make([]SessionListEntry, 0, len(m.sessions))
	for id, e := range m.sessions {
		entries = append(entries, SessionListEntry{
			SessionID: id,
			Info:      e.session.Info(),
		})
	}
	return entries
}

// StopSession stops the session with the given ID.
func (m *SessionManager) StopSession(ctx context.Context, id string) error {
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
func (m *SessionManager) ListDrivers() []driver.Capabilities {
	caps := make([]driver.Capabilities, 0, len(m.drivers))
	for _, d := range m.drivers {
		caps = append(caps, d.Capabilities())
	}
	return caps
}

// forwardStatusEvents reads from the status channel and emits SessionEvent_StatusChange
// events so the assembler flushes buffered text/thought on status transitions.
func (m *SessionManager) forwardStatusEvents(sessionID string, entry *sessionEntry, ch <-chan v2.SessionStatus) {
	for status := range ch {
		seq := entry.nextSeq.Add(1)
		event := &workerv1.SessionEvent{
			SessionId: sessionID,
			Sequence:  seq,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Payload: &workerv1.SessionEvent_StatusChange{
				StatusChange: &workerv1.StatusChange{
					Status: sessionStatusToProto(status),
				},
			},
		}
		m.eventQueue.Append(sessionID, event)
		m.notifyEventSubscribers(SessionEventUpdate{SessionID: sessionID, Event: event})
	}
}

// sessionStatusToProto maps a v2.SessionStatus to the proto enum.
func sessionStatusToProto(s v2.SessionStatus) workerv1.SessionStatus {
	switch s {
	case v2.SessionStatusStarting:
		return workerv1.SessionStatus_SESSION_STATUS_STARTING
	case v2.SessionStatusRunning:
		return workerv1.SessionStatus_SESSION_STATUS_RUNNING
	case v2.SessionStatusIdle:
		return workerv1.SessionStatus_SESSION_STATUS_IDLE
	case v2.SessionStatusStopped:
		return workerv1.SessionStatus_SESSION_STATUS_STOPPED
	case v2.SessionStatusErrored:
		return workerv1.SessionStatus_SESSION_STATUS_ERRORED
	default:
		return workerv1.SessionStatus_SESSION_STATUS_UNSPECIFIED
	}
}

func (m *SessionManager) removeSession(id string) {
	m.mu.Lock()
	_, existed := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if existed {
		m.eventQueue.Remove(id)
		m.notifySubscribers(StateEvent{Type: StateEventRemoved, SessionID: id})
	}
}

// Subscribe returns a channel that receives a StateEvent on every state
// change. The caller must eventually call Unsubscribe.
func (m *SessionManager) Subscribe() chan StateEvent {
	ch := make(chan StateEvent, 16)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (m *SessionManager) Unsubscribe(ch chan StateEvent) {
	m.mu.Lock()
	delete(m.subscribers, ch)
	m.mu.Unlock()
}

func (m *SessionManager) notifySubscribers(event StateEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for ch := range m.subscribers {
		select {
		case ch <- event:
		default: // channel full, drop oldest and send new
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- event:
			default:
			}
		}
	}
}

// SubscribeEvents returns a channel that receives session events.
func (m *SessionManager) SubscribeEvents() chan SessionEventUpdate {
	ch := make(chan SessionEventUpdate, 64)
	m.mu.Lock()
	m.eventSubscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

// UnsubscribeEvents removes an event subscriber channel.
func (m *SessionManager) UnsubscribeEvents(ch chan SessionEventUpdate) {
	m.mu.Lock()
	delete(m.eventSubscribers, ch)
	m.mu.Unlock()
}

func (m *SessionManager) notifyEventSubscribers(evt SessionEventUpdate) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for ch := range m.eventSubscribers {
		select {
		case ch <- evt:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- evt:
			default:
			}
		}
	}
}

// PendingEvents returns all un-ACKed events for a session after the given sequence.
func (m *SessionManager) PendingEvents(sessionID string, afterSeq int64) []*workerv1.SessionEvent {
	return m.eventQueue.Pending(sessionID, afterSeq)
}

// AllPendingEvents returns all pending events across all sessions.
func (m *SessionManager) AllPendingEvents() map[string][]*workerv1.SessionEvent {
	return m.eventQueue.AllPending()
}

// AckEvents drops all events up to the given sequence for a session.
func (m *SessionManager) AckEvents(sessionID string, sequence int64) {
	m.eventQueue.Ack(sessionID, sequence)
}

// emitSessionEvent converts an ACP notification to a proto SessionEvent and enqueues it.
func (m *SessionManager) emitSessionEvent(sessionID string, entry *sessionEntry, n acp.SessionNotification) {
	u := n.Update
	seq := entry.nextSeq.Add(1)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	event := &workerv1.SessionEvent{
		SessionId: sessionID,
		Sequence:  seq,
		Timestamp: now,
	}

	switch {
	case u.AgentMessageChunk != nil:
		text := ""
		if u.AgentMessageChunk.Content.Text != nil {
			text = u.AgentMessageChunk.Content.Text.Text
		}
		event.Payload = &workerv1.SessionEvent_AgentMessageChunk{
			AgentMessageChunk: &workerv1.AgentMessageChunk{Text: text},
		}
	case u.AgentThoughtChunk != nil:
		text := ""
		if u.AgentThoughtChunk.Content.Text != nil {
			text = u.AgentThoughtChunk.Content.Text.Text
		}
		event.Payload = &workerv1.SessionEvent_AgentThoughtChunk{
			AgentThoughtChunk: &workerv1.AgentThoughtChunk{Text: text},
		}
	case u.ToolCall != nil:
		event.Payload = &workerv1.SessionEvent_ToolCallStart{
			ToolCallStart: acpToolCallToProto(u.ToolCall),
		}
	case u.ToolCallUpdate != nil:
		event.Payload = &workerv1.SessionEvent_ToolCallUpdate{
			ToolCallUpdate: acpToolCallUpdateToProto(u.ToolCallUpdate),
		}
	case u.CurrentModeUpdate != nil:
		event.Payload = &workerv1.SessionEvent_ModeChange{
			ModeChange: &workerv1.ModeChange{ModeId: string(u.CurrentModeUpdate.CurrentModeId)},
		}
	default:
		return // skip events we don't handle
	}

	m.eventQueue.Append(sessionID, event)
	m.notifyEventSubscribers(SessionEventUpdate{SessionID: sessionID, Event: event})
}

// SetSessionMode changes the permission mode of a running session.
func (m *SessionManager) SetSessionMode(ctx context.Context, sessionID string, mode driver.SessionMode) error {
	m.mu.RLock()
	e, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	return e.session.SetSessionMode(ctx, mode)
}

// HandleSetTopic updates the topic for the given session and notifies subscribers.
func (m *SessionManager) HandleSetTopic(_ context.Context, sessionID, topic string) error {
	m.mu.Lock()
	e, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	e.topic = topic
	snap := SessionSnapshot{SessionID: sessionID, Info: e.session.Info(), Topic: topic}
	m.mu.Unlock()
	m.notifySubscribers(StateEvent{Type: StateEventUpdate, SessionID: sessionID, Snapshot: &snap})
	m.log.Info("topic set", "session_id", sessionID, "topic", topic)
	return nil
}

// SessionSnapshot is the state of a single session for state sync.
type SessionSnapshot struct {
	SessionID string
	Info      v2.SessionInfo
	Topic     string
}

// HandleHookEvent is a no-op stub for the agentctl EventHandler interface.
// V2 drivers use ACP instead of hooks.
func (m *SessionManager) HandleHookEvent(_ context.Context, event driver.HookEvent) error {
	m.log.Debug("hook event received (no-op in V2)", "session_id", event.SessionID, "hook", event.HookName)
	return nil
}

// HandleStatusReport is a no-op stub for the agentctl EventHandler interface.
func (m *SessionManager) HandleStatusReport(_ context.Context, sessionID, agent, status string) error {
	m.log.Info("status report", "session_id", sessionID, "agent", agent, "status", status)
	return nil
}

// HandlePlanSubmission is a no-op stub for the agentctl EventHandler interface.
func (m *SessionManager) HandlePlanSubmission(_ context.Context, sessionID, agent string, plan []byte) error {
	m.log.Info("plan submitted", "session_id", sessionID, "agent", agent, "plan_bytes", len(plan))
	return nil
}

func logACPEvent(log *slog.Logger, agentID string, n acp.SessionNotification) {
	u := n.Update
	switch {
	case u.AgentMessageChunk != nil:
		text := ""
		if u.AgentMessageChunk.Content.Text != nil {
			text = u.AgentMessageChunk.Content.Text.Text
			if len(text) > 120 {
				text = text[:120] + "..."
			}
		}
		log.Info("acp: agent message", "agent", agentID, "session", n.SessionId, "text", text)
	case u.AgentThoughtChunk != nil:
		text := ""
		if u.AgentThoughtChunk.Content.Text != nil {
			text = u.AgentThoughtChunk.Content.Text.Text
			if len(text) > 120 {
				text = text[:120] + "..."
			}
		}
		log.Info("acp: agent thought", "agent", agentID, "session", n.SessionId, "text", text)
	case u.ToolCall != nil:
		attrs := []any{
			"agent", agentID, "session", n.SessionId,
			"tool_call_id", u.ToolCall.ToolCallId,
			"title", u.ToolCall.Title,
			"status", u.ToolCall.Status,
			"kind", u.ToolCall.Kind,
		}
		for _, loc := range u.ToolCall.Locations {
			attrs = append(attrs, "path", loc.Path)
			if loc.Line != nil {
				attrs = append(attrs, "line", *loc.Line)
			}
		}
		if rawInput := formatRawField(u.ToolCall.RawInput); rawInput != "" {
			attrs = append(attrs, "input", rawInput)
		}
		log.Info("acp: tool call", attrs...)
	case u.ToolCallUpdate != nil:
		attrs := []any{
			"agent", agentID, "session", n.SessionId,
			"tool_call_id", u.ToolCallUpdate.ToolCallId,
		}
		if u.ToolCallUpdate.Status != nil {
			attrs = append(attrs, "status", *u.ToolCallUpdate.Status)
		}
		if u.ToolCallUpdate.Title != nil {
			attrs = append(attrs, "title", *u.ToolCallUpdate.Title)
		}
		for _, loc := range u.ToolCallUpdate.Locations {
			attrs = append(attrs, "path", loc.Path)
			if loc.Line != nil {
				attrs = append(attrs, "line", *loc.Line)
			}
		}
		if rawOutput := formatRawField(u.ToolCallUpdate.RawOutput); rawOutput != "" {
			attrs = append(attrs, "output", rawOutput)
		}
		log.Info("acp: tool call update", attrs...)
	case u.Plan != nil:
		log.Info("acp: plan update", "agent", agentID, "session", n.SessionId, "entries", len(u.Plan.Entries))
	case u.CurrentModeUpdate != nil:
		log.Info("acp: mode update", "agent", agentID, "session", n.SessionId, "mode", u.CurrentModeUpdate.CurrentModeId)
	default:
		log.Info("acp: event", "agent", agentID, "session", n.SessionId)
	}
}

// formatRawField JSON-encodes an any value for logging, truncating to 200 chars.
func formatRawField(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		if len(val) > 200 {
			return val[:200] + "..."
		}
		return val
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		s := string(b)
		if len(s) > 200 {
			return s[:200] + "..."
		}
		return s
	}
}

// CheckSessionResumable checks if an agent driver supports session resume and
// whether the given agent session ID can be loaded.
func (m *SessionManager) CheckSessionResumable(agentID, agentSessionID, cwd string) (bool, string) {
	d, ok := m.drivers[agentID]
	if !ok {
		return false, fmt.Sprintf("unknown agent driver: %s", agentID)
	}

	caps := d.Capabilities()
	if !caps.Has(driver.CapSessionResume) {
		return false, fmt.Sprintf("agent %s does not support session resume", agentID)
	}

	// The driver supports resume — we can't cheaply probe without spawning a full session,
	// so we report resumable=true based on capability alone.
	// The actual LoadSession call during NewSession will fail gracefully if the session
	// file is missing.
	_ = agentSessionID
	_ = cwd
	return true, ""
}

// Prompt sends a follow-up prompt to a running session.
func (m *SessionManager) Prompt(ctx context.Context, sessionID string, blocks []acp.ContentBlock) (*acp.PromptResponse, error) {
	m.mu.RLock()
	e, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return e.session.Prompt(ctx, blocks)
}

// Cancel cancels the active prompt on a running session.
func (m *SessionManager) Cancel(ctx context.Context, sessionID string) error {
	m.mu.RLock()
	e, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	return e.session.Cancel(ctx)
}

// GetStateSnapshot returns the current state of all sessions.
func (m *SessionManager) GetStateSnapshot() []SessionSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := make([]SessionSnapshot, 0, len(m.sessions))
	for id, e := range m.sessions {
		entries = append(entries, SessionSnapshot{
			SessionID: id,
			Info:      e.session.Info(),
			Topic:     e.topic,
		})
	}
	return entries
}

// --- ACP → Proto conversion helpers ---

func acpToolCallToProto(tc *acp.SessionUpdateToolCall) *workerv1.ToolCallStart {
	p := &workerv1.ToolCallStart{
		ToolCallId: string(tc.ToolCallId),
		Title:      tc.Title,
		Kind:       acpToolKindToProto(tc.Kind),
		RawInput:   formatRawField(tc.RawInput),
		Status:     acpToolStatusToProto(tc.Status),
		Content:    acpToolContentToProto(tc.Content),
	}
	for _, loc := range tc.Locations {
		pl := &workerv1.ToolCallLocation{Path: loc.Path}
		if loc.Line != nil {
			pl.Line = int64(*loc.Line)
		}
		p.Locations = append(p.Locations, pl)
	}
	return p
}

func acpToolCallUpdateToProto(tc *acp.SessionToolCallUpdate) *workerv1.ToolCallUpdate {
	p := &workerv1.ToolCallUpdate{
		ToolCallId: string(tc.ToolCallId),
		RawOutput:  formatRawField(tc.RawOutput),
		Content:    acpToolContentToProto(tc.Content),
	}
	if tc.Title != nil {
		p.Title = *tc.Title
	}
	if tc.Status != nil {
		p.Status = acpToolStatusToProto(*tc.Status)
	}
	for _, loc := range tc.Locations {
		pl := &workerv1.ToolCallLocation{Path: loc.Path}
		if loc.Line != nil {
			pl.Line = int64(*loc.Line)
		}
		p.Locations = append(p.Locations, pl)
	}
	return p
}

func acpToolKindToProto(k acp.ToolKind) workerv1.ToolCallKind {
	switch k {
	case "file", "read", "write", "edit":
		return workerv1.ToolCallKind_TOOL_CALL_KIND_FILE
	case "shell", "bash", "command":
		return workerv1.ToolCallKind_TOOL_CALL_KIND_SHELL
	case "search", "grep", "glob":
		return workerv1.ToolCallKind_TOOL_CALL_KIND_SEARCH
	default:
		return workerv1.ToolCallKind_TOOL_CALL_KIND_OTHER
	}
}

func acpToolStatusToProto(s acp.ToolCallStatus) workerv1.ToolCallStatus {
	switch s {
	case "running":
		return workerv1.ToolCallStatus_TOOL_CALL_STATUS_RUNNING
	case "completed":
		return workerv1.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	case "errored":
		return workerv1.ToolCallStatus_TOOL_CALL_STATUS_ERRORED
	default:
		return workerv1.ToolCallStatus_TOOL_CALL_STATUS_RUNNING
	}
}

func acpToolContentToProto(content []acp.ToolCallContent) []*workerv1.ToolCallContentBlock {
	if len(content) == 0 {
		return nil
	}
	blocks := make([]*workerv1.ToolCallContentBlock, 0, len(content))
	for _, c := range content {
		switch {
		case c.Diff != nil:
			b := &workerv1.ToolCallContentBlock{
				Block: &workerv1.ToolCallContentBlock_Diff{
					Diff: &workerv1.ToolCallDiff{
						Path:    c.Diff.Path,
						NewText: c.Diff.NewText,
					},
				},
			}
			if c.Diff.OldText != nil {
				b.GetDiff().OldText = *c.Diff.OldText
			}
			blocks = append(blocks, b)
		case c.Content != nil && c.Content.Content.Text != nil:
			blocks = append(blocks, &workerv1.ToolCallContentBlock{
				Block: &workerv1.ToolCallContentBlock_Text{
					Text: &workerv1.ToolCallText{
						Text: c.Content.Content.Text.Text,
					},
				},
			})
		}
	}
	return blocks
}
