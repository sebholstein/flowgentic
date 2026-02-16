import type { InboxItem } from "@/types/inbox";
import type { ModelInfo } from "@/proto/gen/worker/v1/system_service_pb";
export type { ChatMessage } from "@/lib/session-event-mapper";

export interface ChatTarget {
  type: "thread_overseer" | "task_agent";
  entityId: string;
  agentName: string;
  title: string;
  agentColor?: string;
  /** Current step being worked on (shown with spinner) */
  currentStep?: {
    name: string;
    current: number;
    total: number;
  };
}

export interface AgentChatPanelProps {
  target: ChatTarget;
  onClose?: () => void;
  className?: string;
  /** Hide the header bar */
  hideHeader?: boolean;
  /** Whether to render the setup/config form when chat is empty */
  showSetupOnEmpty?: boolean;
  /** Optional content rendered inside empty state */
  emptyStateContent?: React.ReactNode;
  /** Pending feedback item to display inline */
  pendingFeedback?: InboxItem | null;
  /** Callback when feedback is submitted */
  onFeedbackSubmit?: (itemId: string, data: unknown) => void;
  /** Callback when user sends a message — if provided, replaces mock response logic */
  onSend?: (message: string) => void;
  /** External messages from streaming hook — when provided, replaces internal state */
  externalMessages?: import("@/lib/session-event-mapper").ChatMessage[];
  /** Streaming agent text (not yet finalized into a message) */
  pendingAgentText?: string;
  /** Streaming thought text (not yet finalized into a message) */
  pendingThoughtText?: string;
  /** Whether the stream is actively connected / producing output */
  isStreaming?: boolean;
  /** Currently selected model name */
  selectedModel?: string;
  /** Available models for the dropdown */
  availableModels?: ModelInfo[];
  /** Whether models are still loading */
  modelsLoading?: boolean;
  /** Callback when model changes */
  onModelChange?: (model: string) => void;
  /** Current session mode */
  sessionMode?: string;
  /** Callback when session mode changes */
  onSessionModeChange?: (mode: string) => void;
}
