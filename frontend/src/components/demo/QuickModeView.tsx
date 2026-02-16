import { useState, useCallback, useRef } from "react";
import { AgentChatPanel } from "@/components/chat/AgentChatPanel";
import { FileChangesPanel } from "./FileChangesPanel";
import type { DemoThread } from "@/data/mockFlowgenticData";
import { getDemoMessages } from "@/data/mockFlowgenticData";

export function QuickModeView({ thread }: { thread: DemoThread }) {
  const messages = getDemoMessages(thread.id);
  const agent = thread.agents[0];
  const [leftPanelPercent, setLeftPanelPercent] = useState(45);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startPercent = leftPanelPercent;
      const containerWidth = containerRef.current?.offsetWidth ?? 1;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const deltaPercent = (deltaX / containerWidth) * 100;
        setLeftPanelPercent(Math.min(70, Math.max(25, startPercent + deltaPercent)));
      };

      const handleMouseUp = () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [leftPanelPercent],
  );

  return (
    <div className="flex flex-1 min-h-0" ref={containerRef}>
      {/* Chat panel */}
      <div
        className="flex-shrink-0 h-full overflow-hidden"
        style={{ width: `${leftPanelPercent}%` }}
      >
        <AgentChatPanel
          target={{
            type: "thread_overseer",
            entityId: thread.id,
            agentName: agent.name,
            title: thread.topic,
            agentColor: agent.color,
          }}
          hideHeader
          externalMessages={messages}
        />
      </div>

      {/* Resize handle */}
      <div
        className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
        onMouseDown={handleMouseDown}
      >
        <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors" />
      </div>

      {/* Right pane: file tree + stacked diffs */}
      <div className="min-w-0 flex-1 overflow-hidden">
        <FileChangesPanel />
      </div>
    </div>
  );
}
