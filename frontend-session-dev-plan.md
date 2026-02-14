# Frontend Session Event Integration Plan

## New RPCs Available

### `ListSessionMessages` (Unary)

Returns assembled, complete messages from SQLite.

**Request:**
```protobuf
message ListSessionMessagesRequest {
  string session_id = 1;  // watch a specific session
  string thread_id = 2;   // all sessions for a thread
  string task_id = 3;     // sessions for a specific task
}
```

**Response:**
```protobuf
message ListSessionMessagesResponse {
  repeated SessionMessage messages = 1;
}
```

**`SessionMessage` shape:**
```protobuf
message SessionMessage {
  int64 id = 1;
  string session_id = 2;
  int64 sequence = 3;
  SessionMessageType type = 4;  // 1=agent_message, 2=agent_thought, 3=tool_call, 4=mode_change
  string content = 5;           // JSON-encoded payload (see below)
  string created_at = 6;
}
```

**Content JSON by type:**
- `agent_message` (1): `{"text": "full agent message text"}`
- `agent_thought` (2): `{"text": "full thought text"}`
- `tool_call` (3): `{"tool_call_id":"...","title":"...","kind":1,"raw_input":"...","raw_output":"...","status":2,"locations":[{"path":"...","line":42}]}`
- `mode_change` (4): `{"mode_id": "code"}`

### `WatchSessionEvents` (Server Stream)

Streams history first (as `SessionMessage`), then live events (as `SessionEvent`).

**Request:**
```protobuf
message WatchSessionEventsRequest {
  string session_id = 1;      // watch a specific session
  string thread_id = 2;       // watch all sessions for a thread (most common)
  string task_id = 3;         // watch sessions for a specific task
  int64 after_sequence = 4;   // 0 = include all history
}
```

**Response (stream):**
```protobuf
message WatchSessionEventsResponse {
  oneof update {
    SessionEvent event = 1;       // live chunk from pub-sub
    SessionMessage message = 2;   // assembled message from DB history
  }
}
```

## How to Use from the Thread Route

The most common pattern is watching by `thread_id` from the thread route:

```typescript
const stream = sessionService.watchSessionEvents({
  threadId: threadId,
  afterSequence: 0n,  // get all history
});
```

## Event Types and Their Meaning

### History Messages (via `message` oneof)
These are **complete, assembled messages** from SQLite. They arrive first during the history catch-up phase.

### Live Events (via `event` oneof)
These are **raw streaming chunks** that arrive in real-time:

| Event | Meaning |
|-------|---------|
| `agent_message_chunk` | Partial text of agent's response (accumulate for display) |
| `agent_thought_chunk` | Partial text of agent's thinking (accumulate for display) |
| `tool_call_start` | Agent started a tool call (show tool execution UI) |
| `tool_call_update` | Tool call status/output changed (update tool UI, check for completion) |
| `status_change` | Session status changed (idle, running, stopped) |
| `mode_change` | Session mode changed (e.g., "code" -> "architect") |

### `SessionEvent` shape:
```protobuf
message SessionEvent {
  string session_id = 1;
  int64 sequence = 2;
  string timestamp = 3;
  oneof payload {
    AgentMessageChunk agent_message_chunk = 10;
    AgentThoughtChunk agent_thought_chunk = 11;
    ToolCallStart tool_call_start = 12;
    ToolCallUpdate tool_call_update = 13;
    StatusChange status_change = 14;
    ModeChange mode_change = 15;
  }
}
```

## Mapping to AgentChatPanel Message Types

| Backend Type | Frontend Mapping |
|---|---|
| `SessionMessage` (type=1, agent_message) | Agent text message bubble |
| `SessionMessage` (type=2, agent_thought) | Collapsible thinking section |
| `SessionMessage` (type=3, tool_call) | Tool execution card with title, status, locations |
| `SessionMessage` (type=4, mode_change) | System info message (mode changed) |
| `SessionEvent` (agent_message_chunk) | Append to in-progress agent message |
| `SessionEvent` (agent_thought_chunk) | Append to in-progress thought |
| `SessionEvent` (tool_call_start) | Show new tool card in "running" state |
| `SessionEvent` (tool_call_update) | Update tool card (completed/errored + output) |

## Handling the `WatchSessionEventsResponse` Oneof

```typescript
for await (const resp of stream) {
  if (resp.update.case === "message") {
    // History message from SQLite — render as complete message
    const msg = resp.update.value;
    addCompleteMessage(msg);
  } else if (resp.update.case === "event") {
    // Live event — handle incrementally
    const event = resp.update.value;
    handleLiveEvent(event);
  }
}
```

## Example React Hook Pattern

```typescript
function useSessionEvents(threadId: string) {
  const [messages, setMessages] = useState<SessionMessage[]>([]);
  const [pendingText, setPendingText] = useState("");
  const [pendingThought, setPendingThought] = useState("");
  const [activeTools, setActiveTools] = useState<Map<string, ToolCallStart>>(new Map());

  useEffect(() => {
    const abortController = new AbortController();

    async function watch() {
      const stream = sessionService.watchSessionEvents(
        { threadId, afterSequence: 0n },
        { signal: abortController.signal }
      );

      for await (const resp of stream) {
        if (resp.update.case === "message") {
          setMessages(prev => [...prev, resp.update.value]);
        } else if (resp.update.case === "event") {
          const event = resp.update.value;
          switch (event.payload.case) {
            case "agentMessageChunk":
              setPendingText(prev => prev + event.payload.value.text);
              break;
            case "agentThoughtChunk":
              setPendingThought(prev => prev + event.payload.value.text);
              break;
            case "toolCallStart":
              setActiveTools(prev => new Map(prev).set(
                event.payload.value.toolCallId,
                event.payload.value
              ));
              break;
            case "toolCallUpdate":
              // Update or remove from active tools
              break;
          }
        }
      }
    }

    watch().catch(console.error);
    return () => abortController.abort();
  }, [threadId]);

  return { messages, pendingText, pendingThought, activeTools };
}
```

## Notes on the "Gap"

In-progress message chunks that were sent **before** the user connected to `WatchSessionEvents` will not be visible as live events. However:

1. Once the agent finishes a message, it gets assembled and persisted to SQLite
2. On next `WatchSessionEvents` call, the assembled message appears in history
3. For V1, this gap is acceptable — the user may miss partial streaming of an in-progress message but will see it once complete

**Workaround for future:** Track the last sequence seen and use `after_sequence` to avoid duplicate history on reconnect.
