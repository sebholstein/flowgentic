package session

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

// --- Test doubles ---

type recordingPersister struct {
	events []SessionEvent
}

func (r *recordingPersister) InsertSessionEvent(_ context.Context, evt SessionEvent) error {
	r.events = append(r.events, evt)
	return nil
}

type recordingBroadcaster struct {
	events []SessionEventUpdate
}

func (r *recordingBroadcaster) BroadcastEvent(evt SessionEventUpdate) {
	r.events = append(r.events, evt)
}

func newTestHandler(persister *recordingPersister, broadcaster *recordingBroadcaster) *stateSyncHandler {
	return &stateSyncHandler{
		log:           slog.Default(),
		persister:     persister,
		broadcaster:   broadcaster,
		pendingChunks: make(map[string]*chunkAccumulator),
	}
}

func makeMessageChunk(sessionID, text string, seq int64) *workerv1.SessionEvent {
	return &workerv1.SessionEvent{
		SessionId: sessionID,
		Sequence:  seq,
		Timestamp: "2024-01-01T00:00:00Z",
		Payload: &workerv1.SessionEvent_AgentMessageChunk{
			AgentMessageChunk: &workerv1.AgentMessageChunk{Text: text},
		},
	}
}

func makeThoughtChunk(sessionID, text string, seq int64) *workerv1.SessionEvent {
	return &workerv1.SessionEvent{
		SessionId: sessionID,
		Sequence:  seq,
		Timestamp: "2024-01-01T00:00:00Z",
		Payload: &workerv1.SessionEvent_AgentThoughtChunk{
			AgentThoughtChunk: &workerv1.AgentThoughtChunk{Text: text},
		},
	}
}

func makeToolCall(sessionID string, seq int64) *workerv1.SessionEvent {
	return &workerv1.SessionEvent{
		SessionId: sessionID,
		Sequence:  seq,
		Timestamp: "2024-01-01T00:00:00Z",
		Payload: &workerv1.SessionEvent_ToolCall{
			ToolCall: &workerv1.ToolCall{
				ToolCallId: "tc-1",
				Title:      "test tool",
			},
		},
	}
}

func decodePayload(t *testing.T, payload []byte) SessionEventRecord {
	t.Helper()
	var r SessionEventRecord
	require.NoError(t, json.Unmarshal(payload, &r))
	return r
}

// --- Tests ---

func TestChunkMerging(t *testing.T) {
	t.Run("consecutive message chunks stored as one row", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.HandleSessionEvent("w1", makeMessageChunk("s1", "Hello ", 1))
		h.HandleSessionEvent("w1", makeMessageChunk("s1", "world", 2))
		h.HandleSessionEvent("w1", makeMessageChunk("s1", "!", 3))

		// Nothing persisted yet — chunks are buffered.
		assert.Empty(t, persister.events)

		// Flush triggers the merged write.
		h.FlushAll()

		require.Len(t, persister.events, 1)
		rec := decodePayload(t, persister.events[0].Payload)
		assert.Equal(t, "agent_message_chunk", rec.Type)
		assert.Equal(t, "Hello world!", rec.Text)
		assert.Equal(t, int64(3), rec.Sequence)
	})

	t.Run("consecutive thought chunks stored as one row", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.HandleSessionEvent("w1", makeThoughtChunk("s1", "thinking ", 1))
		h.HandleSessionEvent("w1", makeThoughtChunk("s1", "hard", 2))
		h.FlushAll()

		require.Len(t, persister.events, 1)
		rec := decodePayload(t, persister.events[0].Payload)
		assert.Equal(t, "agent_thought_chunk", rec.Type)
		assert.Equal(t, "thinking hard", rec.Text)
		assert.Equal(t, int64(2), rec.Sequence)
	})

	t.Run("non-chunk event triggers flush", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.HandleSessionEvent("w1", makeMessageChunk("s1", "Hello ", 1))
		h.HandleSessionEvent("w1", makeMessageChunk("s1", "world", 2))

		assert.Empty(t, persister.events)

		// Tool call flushes the pending message chunks, then persists itself.
		h.HandleSessionEvent("w1", makeToolCall("s1", 3))

		require.Len(t, persister.events, 2)

		// First: merged message chunk.
		rec0 := decodePayload(t, persister.events[0].Payload)
		assert.Equal(t, "agent_message_chunk", rec0.Type)
		assert.Equal(t, "Hello world", rec0.Text)
		assert.Equal(t, int64(2), rec0.Sequence)

		// Second: tool call.
		rec1 := decodePayload(t, persister.events[1].Payload)
		assert.Equal(t, "tool_call", rec1.Type)
	})

	t.Run("different chunk types trigger flush", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.HandleSessionEvent("w1", makeMessageChunk("s1", "msg", 1))
		h.HandleSessionEvent("w1", makeThoughtChunk("s1", "thought", 2))

		// Switching from message to thought flushes the message chunk.
		require.Len(t, persister.events, 1)
		rec := decodePayload(t, persister.events[0].Payload)
		assert.Equal(t, "agent_message_chunk", rec.Type)
		assert.Equal(t, "msg", rec.Text)

		// Thought is still buffered.
		h.FlushAll()
		require.Len(t, persister.events, 2)
		rec1 := decodePayload(t, persister.events[1].Payload)
		assert.Equal(t, "agent_thought_chunk", rec1.Type)
		assert.Equal(t, "thought", rec1.Text)
	})

	t.Run("different sessions accumulate independently", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.HandleSessionEvent("w1", makeMessageChunk("s1", "aaa", 1))
		h.HandleSessionEvent("w1", makeMessageChunk("s2", "bbb", 1))
		h.HandleSessionEvent("w1", makeMessageChunk("s1", "AAA", 2))
		h.HandleSessionEvent("w1", makeMessageChunk("s2", "BBB", 2))

		h.FlushAll()

		require.Len(t, persister.events, 2)

		// Find events by session ID.
		bySession := map[string]SessionEventRecord{}
		for _, evt := range persister.events {
			rec := decodePayload(t, evt.Payload)
			bySession[evt.SessionID] = rec
		}

		assert.Equal(t, "aaaAAA", bySession["s1"].Text)
		assert.Equal(t, "bbbBBB", bySession["s2"].Text)
	})

	t.Run("live broadcasts send individual chunks", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.HandleSessionEvent("w1", makeMessageChunk("s1", "a", 1))
		h.HandleSessionEvent("w1", makeMessageChunk("s1", "b", 2))
		h.HandleSessionEvent("w1", makeMessageChunk("s1", "c", 3))

		// Each chunk is broadcast individually.
		require.Len(t, broadcaster.events, 3)
		assert.Equal(t, "s1", broadcaster.events[0].SessionID)

		// But nothing persisted yet.
		assert.Empty(t, persister.events)
	})

	t.Run("FlushAll with no pending chunks is a no-op", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.FlushAll()
		assert.Empty(t, persister.events)
	})

	t.Run("non-chunk event with no pending chunks persists directly", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		h.HandleSessionEvent("w1", makeToolCall("s1", 1))

		require.Len(t, persister.events, 1)
		rec := decodePayload(t, persister.events[0].Payload)
		assert.Equal(t, "tool_call", rec.Type)
	})

	t.Run("merged event uses timestamp from first chunk", func(t *testing.T) {
		persister := &recordingPersister{}
		broadcaster := &recordingBroadcaster{}
		h := newTestHandler(persister, broadcaster)

		evt1 := makeMessageChunk("s1", "a", 1)
		evt1.Timestamp = "2024-01-01T00:00:01Z"
		evt2 := makeMessageChunk("s1", "b", 2)
		evt2.Timestamp = "2024-01-01T00:00:02Z"

		h.HandleSessionEvent("w1", evt1)
		h.HandleSessionEvent("w1", evt2)
		h.FlushAll()

		require.Len(t, persister.events, 1)
		rec := decodePayload(t, persister.events[0].Payload)
		assert.Equal(t, "2024-01-01T00:00:01Z", rec.Timestamp)
	})
}
