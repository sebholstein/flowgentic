import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
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
  const viewLabel = activeView === "threads" ? "Threads" : "Plan Templates";

  return (
    <>
      <div className="flex flex-col gap-2 border-b border-sidebar-border px-3 pb-2.5 pt-2.5">
        <div className="flex items-center justify-between">
          <span className="text-sm font-semibold text-foreground tracking-tight">{viewLabel}</span>
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
