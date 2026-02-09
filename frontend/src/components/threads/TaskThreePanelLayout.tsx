import type { ReactNode } from "react";

interface TaskThreePanelLayoutProps {
  leftPanel: ReactNode;
  rightPanel: ReactNode;
  leftPanelWidth: number;
  onLeftPanelResize: (width: number) => void;
  minLeftWidth?: number;
  maxLeftWidth?: number;
}

export function TaskThreePanelLayout({
  leftPanel,
  rightPanel,
  leftPanelWidth,
  onLeftPanelResize,
  minLeftWidth = 280,
  maxLeftWidth = 500,
}: TaskThreePanelLayoutProps) {
  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = leftPanelWidth;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const newWidth = startWidth + (moveEvent.clientX - startX);
      onLeftPanelResize(Math.min(maxLeftWidth, Math.max(minLeftWidth, newWidth)));
    };

    const handleMouseUp = () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  };

  return (
    <div className="flex h-full min-h-0">
      {/* Left panel: Chat */}
      <div className="flex-shrink-0 overflow-hidden border-r" style={{ width: leftPanelWidth }}>
        {leftPanel}
      </div>

      {/* Resize handle - wide hit area, thin visual line */}
      <div
        className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
        onMouseDown={handleMouseDown}
      >
        <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors pointer-events-none" />
      </div>

      {/* Right panel: File tree + Diff viewer */}
      <div className="min-w-0 flex-1 overflow-hidden">{rightPanel}</div>
    </div>
  );
}
