import type {
  SessionEvent,
  ToolCall,
  ToolCallUpdate,
} from "@/proto/gen/controlplane/v1/session_service_pb";
import { ToolCallKind, ToolCallStatus } from "@/proto/gen/controlplane/v1/session_service_pb";
import type { ToolType, ToolStatus } from "@/components/chat/ToolCallBlock";

/**
 * ChatMessage types matching AgentChatPanel's internal types.
 * Re-exported here so the hook and panel share the same shape.
 */

interface BaseMessage {
  id: string;
  timestamp: string;
}

export interface UserMessage extends BaseMessage {
  type: "user";
  content: string;
}

export interface AgentMessage extends BaseMessage {
  type: "agent";
  content: string;
}

export interface ToolMessage extends BaseMessage {
  type: "tool";
  tool: {
    id: string;
    type: ToolType;
    status: ToolStatus;
    title: string;
    subtitle?: string;
    input?: Record<string, unknown>;
    output?: string;
    content?: string;
    error?: string;
    duration?: number;
  };
}

export interface ThinkingMessage extends BaseMessage {
  type: "thinking";
  thinking: {
    id: string;
    status: "thinking" | "complete";
    content?: string;
    streamingContent?: string;
    duration?: number;
  };
}

export type ChatMessage = UserMessage | AgentMessage | ToolMessage | ThinkingMessage;

// --- Enum mappers ---

export function mapToolCallKind(kind: ToolCallKind): ToolType {
  switch (kind) {
    case ToolCallKind.READ:
    case ToolCallKind.EDIT:
    case ToolCallKind.DELETE:
    case ToolCallKind.MOVE:
      return "edit_file";
    case ToolCallKind.EXECUTE:
      return "bash";
    case ToolCallKind.SEARCH:
      return "search";
    default:
      return "bash";
  }
}

export function mapToolCallStatus(status: ToolCallStatus): ToolStatus {
  switch (status) {
    case ToolCallStatus.IN_PROGRESS:
      return "running";
    case ToolCallStatus.COMPLETED:
      return "success";
    case ToolCallStatus.FAILED:
      return "error";
    default:
      return "running";
  }
}

// --- Live event mappers ---

export function mapToolCallStartToMessage(tc: ToolCall, timestamp: string): ToolMessage {
  const parsedInput = tc.rawInput ? safeParseJSON<Record<string, unknown>>(tc.rawInput) : undefined;
  const toolType = isMcpToolCall(tc.title, parsedInput ?? undefined)
    ? "mcp"
    : mapToolCallKind(tc.kind);
  return {
    id: `tool-${tc.toolCallId}`,
    type: "tool",
    timestamp,
    tool: {
      id: tc.toolCallId,
      type: toolType,
      status: mapToolCallStatus(tc.status),
      title: tc.title,
      subtitle: formatToolSubtitle(tc.kind, tc.rawInput, tc.title),
      input: parsedInput ?? undefined,
      content: extractWrittenContent(tc.kind, tc.rawInput),
    },
  };
}

export function applyToolCallUpdate(existing: ToolMessage, update: ToolCallUpdate): ToolMessage {
  return {
    ...existing,
    tool: {
      ...existing.tool,
      title: update.title || existing.tool.title,
      status: mapToolCallStatus(update.status),
      output: extractOutputText(update.rawOutput) || existing.tool.output,
    },
  };
}

// --- Live event type narrowing helpers ---

export function isAgentMessageChunk(
  evt: SessionEvent,
): evt is SessionEvent & { payload: { case: "agentMessageChunk" } } {
  return evt.payload.case === "agentMessageChunk";
}

export function isAgentThoughtChunk(
  evt: SessionEvent,
): evt is SessionEvent & { payload: { case: "agentThoughtChunk" } } {
  return evt.payload.case === "agentThoughtChunk";
}

export function isToolCallStart(
  evt: SessionEvent,
): evt is SessionEvent & { payload: { case: "toolCall" } } {
  return evt.payload.case === "toolCall";
}

export function isToolCallUpdate(
  evt: SessionEvent,
): evt is SessionEvent & { payload: { case: "toolCallUpdate" } } {
  return evt.payload.case === "toolCallUpdate";
}

export function isStatusChange(
  evt: SessionEvent,
): evt is SessionEvent & { payload: { case: "statusChange" } } {
  return evt.payload.case === "statusChange";
}

export function isUserMessage(
  evt: SessionEvent,
): evt is SessionEvent & { payload: { case: "userMessage" } } {
  return evt.payload.case === "userMessage";
}

// --- Helpers ---

function safeParseJSON<T>(raw: string): T | null {
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

function formatToolSubtitle(
  kind: ToolCallKind,
  rawInput: string | undefined,
  title?: string,
): string | undefined {
  if (rawInput) {
    const input = safeParseJSON<Record<string, unknown>>(rawInput);
    if (input) {
      switch (kind) {
        case ToolCallKind.READ:
        case ToolCallKind.EDIT:
        case ToolCallKind.DELETE:
        case ToolCallKind.MOVE: {
          if (typeof input.filePath === "string") {
            const parts = input.filePath.split("/");
            return parts[parts.length - 1] || input.filePath;
          }
          if (typeof input.path === "string") {
            const parts = input.path.split("/");
            return parts[parts.length - 1] || input.path;
          }
          break;
        }
        case ToolCallKind.EXECUTE: {
          if (typeof input.command === "string") {
            return input.command.length > 60 ? input.command.slice(0, 60) + "..." : input.command;
          }
          if (typeof input.description === "string") {
            return input.description;
          }
          break;
        }
        case ToolCallKind.SEARCH: {
          if (typeof input.pattern === "string") {
            return `pattern: ${input.pattern}`;
          }
          break;
        }
      }
    }
  }

  if (title) {
    if (
      kind === ToolCallKind.READ ||
      kind === ToolCallKind.EDIT ||
      kind === ToolCallKind.DELETE ||
      kind === ToolCallKind.MOVE
    ) {
      const match = title.match(/^(?:Read|Write|Edit|Create|Delete|Move)\s+(.+)$/i);
      if (match) {
        const parts = match[1].split("/");
        return parts[parts.length - 1] || match[1];
      }
    }
    if (kind === ToolCallKind.EXECUTE) {
      const colonIdx = title.indexOf(":");
      if (colonIdx > 0) {
        return title.slice(colonIdx + 1).trim();
      }
    }
    if (kind === ToolCallKind.SEARCH) {
      const match = title.match(/^(?:Search|Find|Grep)\s+(?:for\s+)?(.+)$/i);
      if (match) {
        return match[1];
      }
    }
  }

  return undefined;
}

/**
 * Extract plain-text output from raw_output, which may be a JSON envelope
 * like {"metadata":{...},"output":"actual text"} or just plain text.
 */
function extractOutputText(rawOutput: string | undefined): string | undefined {
  if (!rawOutput) return undefined;
  const parsed = safeParseJSON<Record<string, unknown>>(rawOutput);
  if (parsed) {
    // Common envelope: { output: "...", metadata: { ... } }
    if (typeof parsed.output === "string") return parsed.output.trimEnd();
    // Fallback: { text: "..." }
    if (typeof parsed.text === "string") return parsed.text.trimEnd();
    // If it's a JSON object but no known text field, show a compact summary
    return rawOutput;
  }
  return rawOutput;
}

function extractWrittenContent(
  kind: ToolCallKind,
  rawInput: string | undefined,
): string | undefined {
  if (!rawInput) return undefined;
  const input = safeParseJSON<Record<string, unknown>>(rawInput);
  if (!input) return undefined;

  if (kind === ToolCallKind.EDIT) {
    if (typeof input.newString === "string") return input.newString;
    if (typeof input.content === "string") return input.content;
  }
  return undefined;
}

function isMcpToolCall(title: string | undefined, input?: Record<string, unknown>): boolean {
  if (!title) return false;
  const normalized = title.trim().toLowerCase();
  if (normalized.startsWith("mcp__")) return true;
  if (normalized.startsWith("flowgentic.")) return true;
  if (normalized.startsWith("agentctl.")) return true;
  if (input && typeof input.server === "string" && typeof input.tool === "string") return true;
  return false;
}
