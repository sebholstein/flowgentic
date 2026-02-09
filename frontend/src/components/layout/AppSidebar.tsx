import { useCallback } from "react";
import { useParams, useRouterState } from "@tanstack/react-router";
import { useSidebarStore } from "@/stores/sidebarStore";
import { ThreadsSidebar } from "@/components/threads/ThreadsSidebar";
import { SettingsSidebarContent } from "@/components/settings/SettingsSidebarContent";
import { SidebarFooter } from "@/components/layout/SidebarFooter";

export function AppSidebar() {
  const width = useSidebarStore((s) => s.width);
  const setWidth = useSidebarStore((s) => s.setWidth);
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const params = useParams({ strict: false });
  const selectedThreadId = (params as { threadId?: string }).threadId ?? null;
  const selectedTaskId = (params as { taskId?: string }).taskId ?? null;

  const isSettings = pathname.startsWith("/app/settings");

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startWidth = width;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        setWidth(startWidth + (moveEvent.clientX - startX));
      };

      const handleMouseUp = () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [width, setWidth],
  );

  return (
    <>
      <div className="flex-shrink-0 overflow-hidden flex flex-col" style={{ width }}>
        <div className="flex-1 min-h-0">
          {isSettings ? (
            <SettingsSidebarContent />
          ) : (
            <ThreadsSidebar
              selectedThreadId={selectedThreadId}
              selectedTaskId={selectedTaskId}
            />
          )}
        </div>
        <SidebarFooter />
      </div>
      {/* Resize handle */}
      <div
        className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
        onMouseDown={handleMouseDown}
      >
        <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors pointer-events-none" />
      </div>
    </>
  );
}
