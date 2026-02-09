import { createFileRoute } from "@tanstack/react-router";
import { AssistantRuntimeProvider, useLocalRuntime } from "@assistant-ui/react";
import { Claude } from "@/components/assistant-ui/claude";

export const Route = createFileRoute("/app/overseer/")({
  component: OverseerPage,
});

function OverseerPage() {
  const runtime = useLocalRuntime({
    chatModel: {
      async *run({ messages, abortSignal }) {
        const lastMessage = messages[messages.length - 1];
        const userText =
          lastMessage?.content
            ?.filter((c) => c.type === "text")
            .map((c) => c.text)
            .join("") ?? "";

        // Simulate streaming response
        const response = `I received your message: "${userText}"\n\nThis is a demo response from the Project Overseer. Connect me to a real AI backend to enable actual conversations.`;

        for (const char of response) {
          if (abortSignal.aborted) return;
          yield { text: char };
          await new Promise((r) => setTimeout(r, 10));
        }
      },
    },
  });

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <Claude />
    </AssistantRuntimeProvider>
  );
}
