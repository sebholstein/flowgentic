import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { PanelLeftClose } from "lucide-react";
import { useSidebarStore } from "@/stores/sidebarStore";
import { WindowDragHeader } from "@/components/layout/WindowDragHeader";
import { useIsMacOS } from "@/hooks/use-electron";
import { noDragStyle } from "@/components/layout/WindowDragRegion";
import type { SidebarTab, SidebarView } from "./sidebar-types";

export function SidebarHeader({
  activeView,
  activeTab,
  onTabChange,
  searchQuery,
  onSearchChange,
}: {
  activeView: SidebarView;
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
}) {
  const hideSidebar = useSidebarStore((s) => s.hide);
  const isMacOS = useIsMacOS();

  const viewLabel = activeView === "threads" ? "Threads" : "Plan Templates";

  return (
    <>
      {isMacOS && <WindowDragHeader />}
      <div className="flex flex-col gap-2.5 border-b border-sidebar-border px-3 pb-3 pt-3">
        <div className="flex items-center justify-between">
          <span className="text-sm font-semibold text-foreground tracking-tight">{viewLabel}</span>
          <button
            type="button"
            onClick={hideSidebar}
            style={noDragStyle}
            className="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
            title="Hide sidebar"
          >
            <PanelLeftClose className="size-3.5" />
          </button>
        </div>

        {activeView === "threads" && (
          <>
            <div className="flex gap-1">
              <button
                type="button"
                onClick={() => onTabChange("threads")}
                className={cn(
                  "px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer",
                  activeTab === "threads"
                    ? "bg-muted text-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                )}
              >
                Browse
              </button>
              <button
                type="button"
                onClick={() => onTabChange("archived")}
                className={cn(
                  "px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer",
                  activeTab === "archived"
                    ? "bg-muted text-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                )}
              >
                Archived
              </button>
            </div>
            <Input
              placeholder="Search threads..."
              className="h-7 text-xs"
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
            />
          </>
        )}
      </div>
    </>
  );
}
