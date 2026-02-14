import { useEffect, useRef, useState, useCallback } from "react";
import { useClient } from "@/lib/connect";
import { SessionService } from "@/proto/gen/controlplane/v1/session_service_pb";
import type { SessionEvent } from "@/proto/gen/controlplane/v1/session_service_pb";
import {
  type ChatMessage,
  type ToolMessage,
  mapToolCallStartToMessage,
  applyToolCallUpdate,
  isAgentMessageChunk,
  isAgentThoughtChunk,
  isToolCallStart,
  isToolCallUpdate,
  isStatusChange,
  isUserMessage,
} from "@/lib/session-event-mapper";

interface UseSessionEventsOptions {
  threadId: string | undefined;
  taskId?: string;
}

interface UseSessionEventsResult {
  messages: ChatMessage[];
  pendingAgentText: string;
  pendingThoughtText: string;
  isConnected: boolean;
  isResponding: boolean;
  hasReceivedUpdate: boolean;
  error: Error | null;
  addOptimisticUserMessage: (text: string) => void;
}

export function useSessionEvents({
  threadId,
  taskId,
}: UseSessionEventsOptions): UseSessionEventsResult {
  const client = useClient(SessionService);

  // Rendered state — updated via rAF batching
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [pendingAgentText, setPendingAgentText] = useState("");
  const [pendingThoughtText, setPendingThoughtText] = useState("");
  const [isConnected, setIsConnected] = useState(false);
  const [isResponding, setIsResponding] = useState(false);
  const [hasReceivedUpdate, setHasReceivedUpdate] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // Mutable accumulators (no re-render per chunk)
  const messagesRef = useRef<ChatMessage[]>([]);
  const pendingAgentRef = useRef("");
  const pendingThoughtRef = useRef("");
  const toolIndexRef = useRef<Map<string, number>>(new Map());
  const rafRef = useRef<number | null>(null);
  const dirtyRef = useRef(false);

  const scheduleFlush = useCallback(() => {
    if (dirtyRef.current && rafRef.current == null) {
      rafRef.current = requestAnimationFrame(() => {
        rafRef.current = null;
        dirtyRef.current = false;
        setMessages([...messagesRef.current]);
        setPendingAgentText(pendingAgentRef.current);
        setPendingThoughtText(pendingThoughtRef.current);
      });
    }
  }, []);

  const markDirty = useCallback(() => {
    dirtyRef.current = true;
    scheduleFlush();
  }, [scheduleFlush]);

  /** Finalize pending agent text into a complete AgentMessage. */
  const finalizePendingAgent = useCallback(() => {
    if (pendingAgentRef.current) {
      messagesRef.current.push({
        id: `agent-${Date.now()}-${messagesRef.current.length}`,
        type: "agent",
        timestamp: new Date().toISOString(),
        content: pendingAgentRef.current,
      });
      pendingAgentRef.current = "";
    }
  }, []);

  /** Finalize pending thought text into a complete ThinkingMessage. */
  const finalizePendingThought = useCallback(() => {
    if (pendingThoughtRef.current) {
      messagesRef.current.push({
        id: `thought-${Date.now()}-${messagesRef.current.length}`,
        type: "thinking",
        timestamp: new Date().toISOString(),
        thinking: {
          id: `thought-${Date.now()}`,
          status: "complete",
          content: pendingThoughtRef.current,
        },
      });
      pendingThoughtRef.current = "";
    }
  }, []);

  const processEvent = useCallback(
    (evt: SessionEvent) => {
      if (isAgentMessageChunk(evt)) {
        setIsResponding(true);
        // If we were accumulating thought text, finalize it first
        finalizePendingThought();
        pendingAgentRef.current += evt.payload.value.text;
        markDirty();
        return;
      }

      if (isAgentThoughtChunk(evt)) {
        setIsResponding(true);
        // If we were accumulating agent text, finalize it first
        finalizePendingAgent();
        pendingThoughtRef.current += evt.payload.value.text;
        markDirty();
        return;
      }

      if (isToolCallStart(evt)) {
        setIsResponding(true);
        finalizePendingAgent();
        finalizePendingThought();
        const tc = evt.payload.value;
        const msg = mapToolCallStartToMessage(tc, evt.timestamp);
        const idx = messagesRef.current.length;
        messagesRef.current.push(msg);
        toolIndexRef.current.set(tc.toolCallId, idx);
        markDirty();
        return;
      }

      if (isToolCallUpdate(evt)) {
        const update = evt.payload.value;
        const idx = toolIndexRef.current.get(update.toolCallId);
        if (idx != null) {
          const existing = messagesRef.current[idx];
          if (existing?.type === "tool") {
            messagesRef.current[idx] = applyToolCallUpdate(existing as ToolMessage, update);
            markDirty();
          }
        }
        return;
      }

      if (isStatusChange(evt)) {
        const status = evt.payload.value.status;
        if (status === "SESSION_STATUS_RUNNING" || status === "SESSION_STATUS_STARTING") {
          setIsResponding(true);
        } else if (
          status === "SESSION_STATUS_IDLE" ||
          status === "SESSION_STATUS_STOPPING" ||
          status === "SESSION_STATUS_STOPPED" ||
          status === "SESSION_STATUS_ERRORED" ||
          status === "SESSION_STATUS_UNSPECIFIED"
        ) {
          setIsResponding(false);
        }
        finalizePendingAgent();
        finalizePendingThought();
        markDirty();
        return;
      }

      if (isUserMessage(evt)) {
        finalizePendingAgent();
        finalizePendingThought();
        messagesRef.current.push({
          id: `user-${evt.sequence}-${messagesRef.current.length}`,
          type: "user",
          timestamp: evt.timestamp,
          content: evt.payload.value.text,
        });
        markDirty();
      }
    },
    [finalizePendingAgent, finalizePendingThought, markDirty],
  );

  useEffect(() => {
    if (!threadId) return;

    // Reset state
    messagesRef.current = [];
    pendingAgentRef.current = "";
    pendingThoughtRef.current = "";
    toolIndexRef.current.clear();
    dirtyRef.current = false;
    setMessages([]);
    setPendingAgentText("");
    setPendingThoughtText("");
    setError(null);
    setIsConnected(false);
    setIsResponding(false);
    setHasReceivedUpdate(false);

    const controller = new AbortController();

    (async () => {
      try {
        const request = taskId
          ? { threadId: "", sessionId: "", taskId, afterSequence: BigInt(0) }
          : { threadId, sessionId: "", taskId: "", afterSequence: BigInt(0) };

        for await (const res of client.watchSessionEvents(request, {
          signal: controller.signal,
        })) {
          setHasReceivedUpdate(true);
          if (!isConnected) setIsConnected(true);

          // Unified: both history and live arrive as SessionEvent.
          if (res.event) {
            processEvent(res.event);
          }
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(err as Error);
        }
      }
    })();

    return () => {
      controller.abort();
      if (rafRef.current != null) {
        cancelAnimationFrame(rafRef.current);
        rafRef.current = null;
      }
    };
  }, [threadId, taskId, client, markDirty, processEvent]);

  const addOptimisticUserMessage = useCallback(
    (text: string) => {
      // Finalize any in-flight agent/thought text before injecting the user message,
      // otherwise follow-up response chunks get concatenated onto the previous response.
      finalizePendingAgent();
      finalizePendingThought();
      messagesRef.current.push({
        id: `user-${Date.now()}-${messagesRef.current.length}`,
        type: "user",
        timestamp: new Date().toISOString(),
        content: text,
      });
      markDirty();
    },
    [finalizePendingAgent, finalizePendingThought, markDirty],
  );

  return {
    messages,
    pendingAgentText,
    pendingThoughtText,
    isConnected,
    isResponding,
    hasReceivedUpdate,
    error,
    addOptimisticUserMessage,
  };
}
