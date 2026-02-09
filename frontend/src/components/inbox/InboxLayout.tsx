import { Outlet, useParams, useNavigate, useSearch } from "@tanstack/react-router";
import { useState } from "react";

import { InboxSidebar } from "@/components/inbox/InboxSidebar";
import { inboxItems } from "@/data/mockInboxData";
import type { ViewMode } from "@/types/inbox";

export function InboxLayout() {
  const params = useParams({ strict: false });
  const navigate = useNavigate();
  const { mode: viewMode = "user" } = useSearch({ from: "/app/inbox" }) as { mode?: ViewMode };
  const selectedItemId = (params as { itemId?: string }).itemId ?? null;
  const [filterType, setFilterType] = useState("all");
  const [sidebarWidth, setSidebarWidth] = useState(320);

  const handleViewModeChange = (mode: ViewMode) => {
    navigate({
      to: "/app/inbox",
      search: { mode },
      replace: true,
    });
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const newWidth = startWidth + (moveEvent.clientX - startX);
      // Clamp between 200px and 600px
      setSidebarWidth(Math.min(600, Math.max(200, newWidth)));
    };

    const handleMouseUp = () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  };

  return (
    <div className="flex h-full">
      <div className="flex-shrink-0 overflow-hidden" style={{ width: sidebarWidth }}>
        <InboxSidebar
          items={inboxItems}
          selectedId={selectedItemId}
          viewMode={viewMode}
          onViewModeChange={handleViewModeChange}
          filterType={filterType}
          onFilterTypeChange={setFilterType}
        />
      </div>
      {/* Resize handle - wide hit area, thin visual line */}
      <div
        className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
        onMouseDown={handleMouseDown}
      >
        <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors pointer-events-none" />
      </div>
      <div className="hidden min-w-0 flex-1 overflow-hidden md:block">
        <Outlet />
      </div>
    </div>
  );
}
