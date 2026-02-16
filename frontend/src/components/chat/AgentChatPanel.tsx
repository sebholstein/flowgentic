import { useState, useRef, useEffect } from "react";
import { cn } from "@/lib/utils";
import { ChatHeader } from "./ChatHeader";
import { ChatMessageList } from "./ChatMessageList";
import { ChatComposer } from "./ChatComposer";
import type { AgentChatPanelProps } from "./chat-types";
import type { ChatMessage } from "@/lib/session-event-mapper";

export type { ChatTarget, ChatMessage } from "./chat-types";

export function AgentChatPanel({
  target,
  onClose,
  className,
  hideHeader = false,
  showSetupOnEmpty = true,
  emptyStateContent,
  pendingFeedback,
  onFeedbackSubmit,
  onSend,
  externalMessages,
  pendingAgentText = "",
  pendingThoughtText = "",
  isStreaming = false,
  selectedModel,
  availableModels,
  modelsLoading = false,
  onModelChange,
  sessionMode = "code",
  onSessionModeChange,
}: AgentChatPanelProps) {
  const [internalMessages, setInternalMessages] = useState<ChatMessage[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const [showStreamingIndicator, setShowStreamingIndicator] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Defer streaming indicator by 200ms to avoid flickering on fast loads
  useEffect(() => {
    if (!isStreaming) {
      setShowStreamingIndicator(false);
      return;
    }
    const timer = setTimeout(() => setShowStreamingIndicator(true), 200);
    return () => clearTimeout(timer);
  }, [isStreaming]);

  const messages = externalMessages ?? internalMessages;

  // Auto-scroll to bottom when new messages arrive or pending text changes
  useEffect(() => {
    requestAnimationFrame(() => {
      if (scrollRef.current) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
      }
    });
  }, [messages, pendingAgentText, pendingThoughtText, showStreamingIndicator]);

  // Reset messages and focus when target changes
  useEffect(() => {
    if (!externalMessages) {
      setInternalMessages([]);
    }
  }, [target.entityId, externalMessages]);

  const handleSend = async (text: string) => {
    const userMessage: ChatMessage = {
      id: `msg-${Date.now()}`,
      type: "user",
      content: text,
      timestamp: new Date().toISOString(),
    };

    if (!externalMessages) {
      setInternalMessages((prev) => [...prev, userMessage]);
    }

    if (onSend) {
      onSend(text);
      return;
    }

    // Fallback: no-op when no onSend provided and using external messages
    if (externalMessages) return;

    setIsTyping(true);
    await new Promise((r) => setTimeout(r, 1000 + Math.random() * 1000));
    const agentResponse: ChatMessage = {
      id: `msg-${Date.now()}`,
      type: "agent",
      content: "Message received.",
      timestamp: new Date().toISOString(),
    };
    setInternalMessages((prev) => [...prev, agentResponse]);
    setIsTyping(false);
  };

  const showTypingIndicator = isTyping || showStreamingIndicator;
  const showEmptyState =
    messages.length === 0 &&
    !pendingFeedback &&
    !pendingAgentText &&
    !pendingThoughtText &&
    !showTypingIndicator &&
    !isStreaming;
  const showSetupForm = showEmptyState && showSetupOnEmpty;

  return (
    <div className={cn("flex h-full flex-col", className)}>
      {!hideHeader && (
        <ChatHeader
          target={target}
          pendingFeedback={pendingFeedback}
          onClose={onClose}
        />
      )}

      <div
        className={cn("flex-1 min-h-0 overflow-y-auto", showSetupForm && "flex flex-col")}
        ref={scrollRef}
      >
        <ChatMessageList
          messages={messages}
          target={target}
          pendingFeedback={pendingFeedback}
          onFeedbackSubmit={onFeedbackSubmit}
          pendingAgentText={pendingAgentText}
          pendingThoughtText={pendingThoughtText}
          showTypingIndicator={showTypingIndicator}
          showSetupForm={showSetupForm}
          emptyStateContent={emptyStateContent}
        />
      </div>

      <ChatComposer
        onSend={handleSend}
        isTyping={isTyping}
        selectedModel={selectedModel}
        availableModels={availableModels}
        modelsLoading={modelsLoading}
        onModelChange={onModelChange}
        sessionMode={sessionMode}
        onSessionModeChange={onSessionModeChange}
      />
    </div>
  );
}
