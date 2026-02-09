import { type ReactNode } from "react";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { X, Settings, Keyboard, Server, Sparkles, Bell, Shield, Network } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { WindowDragHeader } from "../layout/WindowDragHeader";
import { useIsMacOS } from "@/hooks/use-electron";

export interface SettingsNavItem {
  id: string;
  label: string;
  icon: ReactNode;
}

export interface SettingsNavSection {
  title: string;
  items: SettingsNavItem[];
}

const navSections: SettingsNavSection[] = [
  {
    title: "Application",
    items: [
      { id: "general", label: "General", icon: <Settings /> },
      { id: "keybindings", label: "Keybindings", icon: <Keyboard /> },
      { id: "notifications", label: "Notifications", icon: <Bell /> },
    ],
  },
  {
    title: "Backend",
    items: [
      { id: "infrastructure", label: "Control Planes & Workers", icon: <Network /> },
      { id: "providers", label: "Providers", icon: <Server /> },
      { id: "models", label: "Models", icon: <Sparkles /> },
    ],
  },
  {
    title: "Account",
    items: [{ id: "security", label: "Security", icon: <Shield /> }],
  },
];

export function SettingsSidebarContent() {
  const navigate = useNavigate();
  const isMacOS = useIsMacOS();
  const search = useRouterState({
    select: (s) => s.location.search as Record<string, string | undefined>,
  });
  const activeSection = search.section ?? "general";

  const handleSectionChange = (sectionId: string) => {
    navigate({ to: "/app/settings", search: { section: sectionId } });
  };

  const handleClose = () => {
    navigate({ to: "/app/threads" });
  };

  return (
    <div className="flex h-full flex-col bg-sidebar select-none">
      <div className="relative">
        <WindowDragHeader />
        <div className="absolute right-1 top-2 flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClose}
            className="size-6 p-0 text-muted-foreground hover:text-foreground"
          >
            <X className="size-3.5" />
          </Button>
        </div>
      </div>
      <div className={cn("flex items-center px-4 py-2 pb-2", isMacOS && "mt-4")}>
        <span className="text-sm font-medium text-foreground">Settings</span>
      </div>
      <ScrollArea className="flex-1 px-4">
        <nav className="py-2 pl-2">
          {navSections.map((section, sectionIndex) => (
            <div key={section.title} className={cn(sectionIndex > 0 && "mt-4")}>
              <h3 className="-ml-2 mb-1 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                {section.title}
              </h3>
              <ul className="space-y-0.5">
                {section.items.map((item) => (
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => handleSectionChange(item.id)}
                      className={cn(
                        "-ml-2 flex w-full items-center gap-2 rounded-md px-2 py-1 text-left text-[13px] transition-colors",
                        activeSection === item.id
                          ? "bg-primary/10 text-primary"
                          : "text-muted-foreground hover:bg-muted hover:text-foreground",
                      )}
                    >
                      <span className="flex size-3.5 items-center justify-center [&>svg]:size-3.5">
                        {item.icon}
                      </span>
                      <span className="truncate">{item.label}</span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>
      </ScrollArea>
    </div>
  );
}
