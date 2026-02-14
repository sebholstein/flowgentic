import { MainLayout } from "@/components/layout/MainLayout";
import { SidebarWrapper } from "@/components/layout/SidebarWrapper";
import { createFileRoute, Outlet, useRouterState } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { PanelLeft } from "lucide-react";
import { useSidebarStore } from "@/stores/sidebarStore";
import { useHotkeys } from "react-hotkeys-hook";
import { useIsMacOS } from "@/hooks/use-electron";
import { cn } from "@/lib/utils";
import { noDragStyle } from "@/components/layout/WindowDragRegion";

export const Route = createFileRoute("/app")({
  component: RouteComponent,
});

function RouteComponent() {
  const visible = useSidebarStore((s) => s.visible);
  const toggle = useSidebarStore((s) => s.toggle);
  const isMacOS = useIsMacOS();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const isSettings = pathname.startsWith("/app/settings");

  useHotkeys(
    "mod+b",
    (e) => {
      e.preventDefault();
      // Don't allow hiding the sidebar when on settings
      if (isSettings && visible) return;
      toggle();
    },
    { enableOnFormTags: ["INPUT", "TEXTAREA", "SELECT"] },
    [isSettings, visible, toggle],
  );

  return (
    <MainLayout>
      <div className="flex h-full min-h-0 flex-1">
        {visible && <SidebarWrapper />}
        <div
          className={cn(
            "relative min-w-0 flex-1 overflow-auto bg-background",
            !visible && isMacOS && "pl-24",
          )}
        >
          <Outlet context={{ sidebarVisible: visible, isMacOS }} />
          {!visible && (
            <Button
              variant="ghost"
              size="sm"
              onClick={toggle}
              style={noDragStyle}
              className={cn(
                "absolute z-50 size-6 p-0 text-muted-foreground hover:text-foreground",
                isMacOS ? "left-[80px] top-2" : "left-2 top-2",
              )}
            >
              <PanelLeft className="size-3.5" />
            </Button>
          )}
        </div>
      </div>
    </MainLayout>
  );
}
