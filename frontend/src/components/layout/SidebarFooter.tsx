import { Link, useNavigate, useRouterState } from "@tanstack/react-router";
import { Check, ChevronsUpDown, Server, Settings } from "lucide-react";
import { cn } from "@/lib/utils";
import { useInfrastructureStore, selectActiveControlPlane } from "@/stores/serverStore";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const statusDotColors: Record<string, string> = {
  connecting: "bg-amber-500 animate-pulse",
  connected: "bg-emerald-500",
  disconnected: "bg-slate-400",
  error: "bg-red-500",
};

export function SidebarFooter() {
  const navigate = useNavigate();
  const controlPlanes = useInfrastructureStore((s) => s.controlPlanes);
  const activeControlPlaneId = useInfrastructureStore((s) => s.activeControlPlaneId);
  const setActiveControlPlane = useInfrastructureStore((s) => s.setActiveControlPlane);
  const activeControlPlane = useInfrastructureStore(selectActiveControlPlane);
  const workers = useInfrastructureStore((s) => s.workers);
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const isOnSettings = pathname.startsWith("/app/settings");

  return (
    <div className="border-t border-sidebar-border p-2 flex items-center justify-between">
      <Link
        to={isOnSettings ? "/app/threads" : "/app/settings"}
        className={cn(
          "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors",
          isOnSettings && "bg-muted text-foreground",
        )}
      >
        <Settings className="size-4" />
        <span className="text-xs">Settings</span>
      </Link>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors">
            <span className="relative">
              <Server className="size-3.5" />
              <span
                className={cn(
                  "absolute -top-0.5 -right-0.5 size-2 rounded-full",
                  statusDotColors[activeControlPlane.status],
                )}
              />
            </span>
            <span className="text-xs max-w-[120px] truncate">{activeControlPlane.name}</span>
            <ChevronsUpDown className="size-3 shrink-0 text-muted-foreground" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="top" align="end" className="min-w-52">
          <DropdownMenuLabel className="text-xs text-muted-foreground font-normal">
            Control Planes
          </DropdownMenuLabel>
          <DropdownMenuGroup>
            {controlPlanes.map((cp) => {
              const workerCount = workers.filter((w) => w.controlPlaneId === cp.id).length;
              return (
                <DropdownMenuItem
                  key={cp.id}
                  onClick={() => setActiveControlPlane(cp.id)}
                  className="flex items-center gap-2"
                >
                  <span className="relative">
                    <Server className="size-3.5" />
                    <span
                      className={cn(
                        "absolute -top-0.5 -right-0.5 size-1.5 rounded-full",
                        statusDotColors[cp.status],
                      )}
                    />
                  </span>
                  <span className="text-xs">{cp.name}</span>
                  <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-4 font-normal">
                    {workerCount} {workerCount === 1 ? "worker" : "workers"}
                  </Badge>
                  {cp.id === activeControlPlaneId && <Check className="size-3.5 shrink-0" />}
                </DropdownMenuItem>
              );
            })}
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => navigate({ to: "/app/settings", search: { section: "infrastructure" } })}
          >
            <Settings className="size-3.5" />
            <span className="text-xs">Settings</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
