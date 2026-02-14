import { createFileRoute } from "@tanstack/react-router";
import {
  Workflow,
  Command,
  Plus,
  Settings,
  Search,
  MessageSquarePlus,
  LayoutGrid,
} from "lucide-react";
import { useIsMacOS } from "@/hooks/use-electron";

export const Route = createFileRoute("/app/")({
  component: StartPage,
});

function Kbd({ children }: { children: React.ReactNode }) {
  return (
    <kbd className="inline-flex items-center justify-center min-w-5 h-5 px-1 rounded bg-muted border border-border text-[10px] font-medium text-muted-foreground">
      {children}
    </kbd>
  );
}

function ShortcutRow({
  icon: Icon,
  label,
  keys,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  keys: React.ReactNode[];
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-2 px-3 rounded-md hover:bg-muted/50 transition-colors">
      <div className="flex items-center gap-2.5 text-sm text-foreground/80">
        <Icon className="size-3.5 text-muted-foreground" />
        <span>{label}</span>
      </div>
      <div className="flex items-center gap-1">
        {keys.map((key, i) => (
          <Kbd key={i}>{key}</Kbd>
        ))}
      </div>
    </div>
  );
}

function StartPage() {
  const isMac = useIsMacOS();
  const mod = isMac ? "\u2318" : "Ctrl";

  return (
    <div className="flex h-full items-center justify-center">
      <div className="flex flex-col items-center gap-8 max-w-sm w-full px-4">
        {/* Logo */}
        <div className="flex flex-col items-center gap-3">
          <div className="flex items-center justify-center size-14 rounded-2xl bg-primary/10 border border-primary/20">
            <Workflow className="size-7 text-primary" />
          </div>
          <h1 className="text-xl font-semibold tracking-tight">flowgentic</h1>
          <p className="text-xs text-muted-foreground">Agentic workflow orchestration</p>
        </div>

        {/* Shortcuts */}
        <div className="w-full space-y-1">
          <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider px-3 mb-2">
            Keyboard Shortcuts
          </p>
          <ShortcutRow icon={Command} label="Command Palette" keys={[mod, "P"]} />
          <ShortcutRow icon={Plus} label="New Thread" keys={[mod, "N"]} />
          <ShortcutRow icon={Search} label="Search Threads" keys={[mod, "K"]} />
          <ShortcutRow icon={MessageSquarePlus} label="Quick Message" keys={[mod, "Shift", "M"]} />
          <ShortcutRow icon={LayoutGrid} label="Toggle Sidebar" keys={[mod, "B"]} />
          <ShortcutRow icon={Settings} label="Settings" keys={[mod, ","]} />
        </div>
      </div>
    </div>
  );
}
