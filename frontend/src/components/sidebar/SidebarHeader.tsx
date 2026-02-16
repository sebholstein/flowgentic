import { useSetAtom } from "jotai";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  PanelLeftClose,
  MessagesSquare,
  LayoutTemplate,
  Search,
  Clock,
  Star,
  Settings,
} from "lucide-react";
import { commandPaletteOpenAtom } from "@/stores/atoms";
import { useSidebarStore } from "@/stores/sidebarStore";
import { WindowDragHeader } from "@/components/layout/WindowDragHeader";
import { useIsMacOS } from "@/hooks/use-electron";
import type { SidebarTab, SidebarView } from "./sidebar-types";

function NavIconButton({
  icon: Icon,
  title,
  isActive,
  onClick,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  isActive?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "p-1.5 rounded-md transition-colors cursor-pointer flex items-center justify-center",
        isActive
          ? "bg-muted text-foreground"
          : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
      )}
      title={title}
    >
      <Icon className="size-3.5" />
    </button>
  );
}

export function SidebarHeader({
  activeView,
  onViewChange,
  activeTab,
  onTabChange,
  searchQuery,
  onSearchChange,
}: {
  activeView: SidebarView;
  onViewChange: (view: SidebarView) => void;
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
}) {
  const hideSidebar = useSidebarStore((s) => s.hide);
  const setCommandPaletteOpen = useSetAtom(commandPaletteOpenAtom);
  const isMacOS = useIsMacOS();

  return (
    <>
      <div className="relative">
        <WindowDragHeader />
        <Button
          variant="ghost"
          size="sm"
          onClick={hideSidebar}
          className="absolute right-1 top-2 size-6 p-0 text-muted-foreground hover:text-foreground"
        >
          <PanelLeftClose className="size-3.5" />
        </Button>
      </div>
      <div className={cn("flex flex-col gap-2.5 border-b p-4 pt-0 pb-3", isMacOS && "mt-4")}>
        <div className="flex items-center justify-start gap-0.5">
          <NavIconButton icon={MessagesSquare} title="Threads" isActive={activeView === "threads"} onClick={() => onViewChange("threads")} />
          <NavIconButton icon={LayoutTemplate} title="Plan Templates" isActive={activeView === "templates"} onClick={() => onViewChange("templates")} />
          <NavIconButton icon={Search} title="Search" onClick={() => setCommandPaletteOpen(true)} />
          <NavIconButton icon={Clock} title="History" />
          <NavIconButton icon={Star} title="Favorites" />
          <NavIconButton icon={Settings} title="Settings" />
        </div>

        {activeView === "threads" ? (
          <>
            <div className="text-sm font-medium text-foreground">Threads</div>
            <div className="flex justify-between -mt-1">
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
              </div>
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
              className="h-8"
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
            />
          </>
        ) : (
          <div className="text-sm font-medium text-foreground">Plan Templates</div>
        )}
      </div>
    </>
  );
}
