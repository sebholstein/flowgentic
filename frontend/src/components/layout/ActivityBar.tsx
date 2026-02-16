import { useSetAtom } from "jotai";
import { cn } from "@/lib/utils";
import {
  MessagesSquare,
  LayoutTemplate,
  Search,
  Inbox,
} from "lucide-react";
import { commandPaletteOpenAtom } from "@/stores/atoms";
import { dragStyle, noDragStyle } from "@/components/layout/WindowDragRegion";
import { useIsMacOS } from "@/hooks/use-electron";
import type { SidebarView } from "@/components/sidebar/sidebar-types";

function ActivityBarIcon({
  icon: Icon,
  label,
  isActive,
  onClick,
  badge,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  isActive?: boolean;
  onClick?: () => void;
  badge?: number;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={noDragStyle}
      className={cn(
        "relative w-8 h-8 flex items-center justify-center rounded-md transition-colors cursor-pointer group",
        isActive
          ? "text-activity-bar-active bg-activity-bar-active/10"
          : "text-activity-bar-foreground hover:text-activity-bar-active hover:bg-activity-bar-active/5",
      )}
      title={label}
    >
      {isActive && (
        <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[2px] h-4 rounded-r-full bg-primary" />
      )}
      <Icon className="size-4" />
      {badge !== undefined && badge > 0 && (
        <span className="absolute top-1 right-1 min-w-[14px] h-[14px] px-1 text-[9px] font-semibold leading-[14px] text-center rounded-full bg-rose-500 text-white">
          {badge > 99 ? "99+" : badge}
        </span>
      )}
    </button>
  );
}

export function ActivityBar({
  activeView,
  onViewChange,
  inboxCount,
}: {
  activeView: SidebarView;
  onViewChange: (view: SidebarView) => void;
  inboxCount?: number;
}) {
  const setCommandPaletteOpen = useSetAtom(commandPaletteOpenAtom);
  const isMacOS = useIsMacOS();

  return (
    <div
      className="flex flex-col items-center w-10 flex-shrink-0 bg-activity-bar"
      style={dragStyle}
    >
      {/* Main nav — aligned with sidebar title */}
      <div className={cn("flex flex-col items-center gap-0.5", isMacOS ? "mt-10" : "mt-[20px]")}>
        <ActivityBarIcon
          icon={MessagesSquare}
          label="Threads"
          isActive={activeView === "threads"}
          onClick={() => onViewChange("threads")}
        />
        <ActivityBarIcon
          icon={LayoutTemplate}
          label="Templates"
          isActive={activeView === "templates"}
          onClick={() => onViewChange("templates")}
        />
        <ActivityBarIcon
          icon={Search}
          label="Search"
          onClick={() => setCommandPaletteOpen(true)}
        />
        <ActivityBarIcon
          icon={Inbox}
          label="Inbox"
          badge={inboxCount}
        />
      </div>

    </div>
  );
}
