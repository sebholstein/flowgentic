import { useState, useCallback } from "react";
import { useParams, useRouterState } from "@tanstack/react-router";
import { useSidebarStore } from "@/stores/sidebarStore";
import { MainSidebar } from "@/components/sidebar/MainSidebar";
import { SettingsSidebarContent } from "@/components/settings/SettingsSidebarContent";
import { SidebarFooter } from "@/components/layout/SidebarFooter";
import { ActivityBar } from "@/components/layout/ActivityBar";
import type { SidebarView } from "@/components/sidebar/sidebar-types";

export function SidebarWrapper() {
  const width = useSidebarStore((s) => s.width);
  const setWidth = useSidebarStore((s) => s.setWidth);
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const params = useParams({ strict: false });
  const selectedThreadId =
    (params as { threadId?: string }).threadId ??
    (params as { scenarioId?: string }).scenarioId ??
    null;
  const selectedTaskId = (params as { taskId?: string }).taskId ?? null;

  const isSettings = pathname.startsWith("/app/settings");

  const [activeView, setActiveView] = useState<SidebarView>("threads");

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
      <ActivityBar
        activeView={activeView}
        onViewChange={setActiveView}
      />
      <div className="flex-shrink-0 flex flex-col my-1.5" style={{ width }}>
        <div className="flex-1 min-h-0 flex flex-col overflow-hidden bg-sidebar rounded-lg">
          <div className="flex-1 min-h-0">
            {isSettings ? (
              <SettingsSidebarContent />
            ) : (
              <MainSidebar
                selectedThreadId={selectedThreadId}
                selectedTaskId={selectedTaskId}
                activeView={activeView}
              />
            )}
          </div>
          <SidebarFooter />
        </div>
      </div>
      {/* Resize handle */}
      <div
        className="w-2 flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
        onMouseDown={handleMouseDown}
      >
        <div className="w-px h-full bg-transparent group-hover:bg-primary/30 transition-colors pointer-events-none" />
      </div>
    </>
  );
}
