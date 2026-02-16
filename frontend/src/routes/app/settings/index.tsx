import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useSidebarStore } from "@/stores/sidebarStore";
import { Pencil, Trash2, Plus, Bot, RefreshCw, RotateCcw } from "lucide-react";
import { useClient } from "@/lib/connect";
import { SystemService } from "@/proto/gen/worker/v1/system_service_pb";
import { WorkerService } from "@/proto/gen/controlplane/v1/worker_service_pb";
import {
  EmbeddedWorkerService,
  EmbeddedWorkerStatus,
} from "@/proto/gen/controlplane/v1/embedded_worker_service_pb";
import type { WatchEmbeddedWorkerStatusResponse } from "@/proto/gen/controlplane/v1/embedded_worker_service_pb";
import { getHarnessIcon } from "@/components/icons/agent-icons";

import { Badge } from "@/components/ui/badge";
import { SettingsSection } from "@/components/settings/SettingsSection";
import { SettingsItem } from "@/components/settings/SettingsItem";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { useInfrastructureStore } from "@/stores/serverStore";
import { ServerStatusDot } from "@/components/servers/ServerStatusDot";
import type { ConnectionStatus, ControlPlane, Worker } from "@/types/server";
import { dragStyle } from "@/components/layout/WindowDragRegion";

type SettingsSearchParams = {
  section?: string;
};

export const Route = createFileRoute("/app/settings/")({
  component: SettingsPage,
  validateSearch: (search: Record<string, unknown>): SettingsSearchParams => ({
    section: typeof search.section === "string" ? search.section : undefined,
  }),
});

const sectionTitles: Record<string, string> = {
  general: "General",
  keybindings: "Keybindings",
  notifications: "Notifications",
  infrastructure: "Control Planes & Workers",
  providers: "Providers",
  models: "Models",
  security: "Security",
};

function SettingsPage() {
  const { section } = Route.useSearch();
  const activeSection = section ?? "general";
  const sectionTitle = sectionTitles[activeSection] ?? "Settings";
  const showSidebar = useSidebarStore((s) => s.show);

  // Ensure sidebar is visible when on settings
  useEffect(() => {
    showSidebar();
  }, [showSidebar]);

  return (
    <div className="flex h-full flex-col bg-surface rounded-lg select-none overflow-hidden relative">
      {/* Header with drag region */}
      <div className="flex border-b h-10 shrink-0 items-center gap-2 px-4" style={dragStyle}>
        <span className="font-medium text-sm">{sectionTitle}</span>
      </div>

      <ScrollArea className="flex-1 min-h-0">
        <div className="p-6 max-w-2xl">
          {activeSection === "general" && <GeneralSettings />}
          {activeSection === "keybindings" && <KeybindingsSettings />}
          {activeSection === "notifications" && <NotificationsSettings />}
          {activeSection === "infrastructure" && <ControlPlanesAndWorkersSettings />}
          {activeSection === "providers" && <ProvidersSettings />}
          {activeSection === "models" && <ModelsSettings />}
          {activeSection === "security" && <SecuritySettings />}
        </div>
      </ScrollArea>
    </div>
  );
}

function GeneralSettings() {
  const [language, setLanguage] = useState("en");
  const [appearance, setAppearance] = useState("system");
  const [theme, setTheme] = useState("default");

  return (
    <>
      <SettingsSection title="Appearance">
        <SettingsItem label="Language" description="Change the display language for Flowgentic">
          <Select value={language} onValueChange={setLanguage}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="en">English</SelectItem>
              <SelectItem value="de">Deutsch</SelectItem>
              <SelectItem value="es">Español</SelectItem>
              <SelectItem value="fr">Français</SelectItem>
            </SelectContent>
          </Select>
        </SettingsItem>

        <SettingsItem label="Appearance" description="Adjust how Flowgentic looks on your device">
          <Select value={appearance} onValueChange={setAppearance}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="system">System</SelectItem>
              <SelectItem value="light">Light</SelectItem>
              <SelectItem value="dark">Dark</SelectItem>
            </SelectContent>
          </Select>
        </SettingsItem>

        <SettingsItem label="Theme" description="Customize the theme of Flowgentic">
          <Select value={theme} onValueChange={setTheme}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="default">Default</SelectItem>
              <SelectItem value="zinc">Zinc</SelectItem>
              <SelectItem value="slate">Slate</SelectItem>
              <SelectItem value="stone">Stone</SelectItem>
            </SelectContent>
          </Select>
        </SettingsItem>
      </SettingsSection>

      <SettingsSection title="Behavior">
        <SettingsItem
          label="Auto-refresh"
          description="Automatically refresh data when returning to the app"
        >
          <Switch defaultChecked />
        </SettingsItem>

        <SettingsItem
          label="Confirm destructive actions"
          description="Show confirmation dialogs before deleting items"
        >
          <Switch defaultChecked />
        </SettingsItem>
      </SettingsSection>
    </>
  );
}

function KeybindingsSettings() {
  return (
    <>
      <SettingsSection title="Navigation">
        <SettingsItem label="Open command palette" description="Quick access to all commands">
          <kbd className="rounded bg-muted px-2 py-1 text-xs font-mono">⌘K</kbd>
        </SettingsItem>

        <SettingsItem label="Go to inbox" description="Navigate to your inbox">
          <kbd className="rounded bg-muted px-2 py-1 text-xs font-mono">⌘1</kbd>
        </SettingsItem>

        <SettingsItem label="Go to issues" description="Navigate to issues">
          <kbd className="rounded bg-muted px-2 py-1 text-xs font-mono">⌘2</kbd>
        </SettingsItem>

        <SettingsItem label="Go to settings" description="Navigate to settings">
          <kbd className="rounded bg-muted px-2 py-1 text-xs font-mono">⌘,</kbd>
        </SettingsItem>
      </SettingsSection>

      <SettingsSection title="Actions">
        <SettingsItem label="Create new issue" description="Open new issue dialog">
          <kbd className="rounded bg-muted px-2 py-1 text-xs font-mono">⌘N</kbd>
        </SettingsItem>

        <SettingsItem label="Search" description="Search across all items">
          <kbd className="rounded bg-muted px-2 py-1 text-xs font-mono">⌘F</kbd>
        </SettingsItem>
      </SettingsSection>
    </>
  );
}

function NotificationsSettings() {
  return (
    <>
      <SettingsSection title="System Notifications">
        <SettingsItem
          label="Agent updates"
          description="Show notification when an agent completes or needs attention"
        >
          <Switch defaultChecked />
        </SettingsItem>

        <SettingsItem
          label="Permissions required"
          description="Show notification when a permission is required"
        >
          <Switch defaultChecked />
        </SettingsItem>

        <SettingsItem label="Issue status changes" description="Notify when issues change status">
          <Switch />
        </SettingsItem>
      </SettingsSection>

      <SettingsSection title="Email Notifications">
        <SettingsItem label="Daily digest" description="Receive a daily summary of activity">
          <Switch />
        </SettingsItem>

        <SettingsItem label="Critical alerts" description="Email alerts for urgent items">
          <Switch defaultChecked />
        </SettingsItem>
      </SettingsSection>
    </>
  );
}

function ProvidersSettings() {
  const [defaultProvider, setDefaultProvider] = useState("anthropic");

  return (
    <>
      <SettingsSection title="AI Providers" description="Configure your AI service providers">
        <SettingsItem label="Default provider" description="The primary provider used for AI tasks">
          <Select value={defaultProvider} onValueChange={setDefaultProvider}>
            <SelectTrigger className="w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="anthropic">Anthropic</SelectItem>
              <SelectItem value="openai">OpenAI</SelectItem>
              <SelectItem value="local">Local</SelectItem>
            </SelectContent>
          </Select>
        </SettingsItem>

        <SettingsItem
          label="Use fallback"
          description="Automatically switch to backup provider on failure"
        >
          <Switch />
        </SettingsItem>
      </SettingsSection>

      <SettingsSection title="API Keys">
        <SettingsItem label="Anthropic API Key" description="Your Claude API key">
          <span className="text-xs text-muted-foreground">••••••••sk-abc123</span>
        </SettingsItem>

        <SettingsItem label="OpenAI API Key" description="Your OpenAI API key">
          <span className="text-xs text-muted-foreground">Not configured</span>
        </SettingsItem>
      </SettingsSection>
    </>
  );
}

function ModelsSettings() {
  const [defaultModel, setDefaultModel] = useState("claude-opus-4");
  const [fastModel, setFastModel] = useState("claude-sonnet-4");

  return (
    <>
      <SettingsSection
        title="Model Selection"
        description="Choose which models to use for different tasks"
      >
        <SettingsItem label="Default model" description="Used for complex reasoning and planning">
          <Select value={defaultModel} onValueChange={setDefaultModel}>
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="claude-opus-4">Claude Opus 4</SelectItem>
              <SelectItem value="claude-sonnet-4">Claude Sonnet 4</SelectItem>
              <SelectItem value="gpt-4o">GPT-4o</SelectItem>
            </SelectContent>
          </Select>
        </SettingsItem>

        <SettingsItem label="Fast model" description="Used for quick tasks and code generation">
          <Select value={fastModel} onValueChange={setFastModel}>
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="claude-sonnet-4">Claude Sonnet 4</SelectItem>
              <SelectItem value="claude-haiku-3">Claude Haiku 3</SelectItem>
              <SelectItem value="gpt-4o-mini">GPT-4o Mini</SelectItem>
            </SelectContent>
          </Select>
        </SettingsItem>
      </SettingsSection>

      <SettingsSection title="Model Behavior">
        <SettingsItem
          label="Streaming responses"
          description="Show AI responses as they are generated"
        >
          <Switch defaultChecked />
        </SettingsItem>

        <SettingsItem
          label="Extended thinking"
          description="Allow models to think longer for complex problems"
        >
          <Switch defaultChecked />
        </SettingsItem>
      </SettingsSection>
    </>
  );
}

function SecuritySettings() {
  return (
    <>
      <SettingsSection title="Authentication">
        <SettingsItem
          label="Two-factor authentication"
          description="Add an extra layer of security to your account"
        >
          <Switch />
        </SettingsItem>

        <SettingsItem label="Session timeout" description="Automatically log out after inactivity">
          <Select defaultValue="24h">
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">1 hour</SelectItem>
              <SelectItem value="8h">8 hours</SelectItem>
              <SelectItem value="24h">24 hours</SelectItem>
              <SelectItem value="7d">7 days</SelectItem>
            </SelectContent>
          </Select>
        </SettingsItem>
      </SettingsSection>

      <SettingsSection title="Privacy">
        <SettingsItem
          label="Usage analytics"
          description="Help improve Flowgentic by sharing anonymous usage data"
        >
          <Switch />
        </SettingsItem>

        <SettingsItem
          label="Conversation history"
          description="Store conversation history for context"
        >
          <Switch defaultChecked />
        </SettingsItem>
      </SettingsSection>
    </>
  );
}

function ControlPlanesAndWorkersSettings() {
  const controlPlanes = useInfrastructureStore((s) => s.controlPlanes);
  const activeControlPlaneId = useInfrastructureStore((s) => s.activeControlPlaneId);
  const setActiveControlPlane = useInfrastructureStore((s) => s.setActiveControlPlane);
  const removeControlPlane = useInfrastructureStore((s) => s.removeControlPlane);
  const addControlPlane = useInfrastructureStore((s) => s.addControlPlane);
  const updateControlPlane = useInfrastructureStore((s) => s.updateControlPlane);
  const addWorker = useInfrastructureStore((s) => s.addWorker);
  const removeWorker = useInfrastructureStore((s) => s.removeWorker);
  const updateWorker = useInfrastructureStore((s) => s.updateWorker);
  const allWorkers = useInfrastructureStore((s) => s.workers);

  const embeddedCP = controlPlanes.find((c) => c.type === "embedded");
  const embeddedWorkers = allWorkers.filter((w) => w.controlPlaneId === embeddedCP?.id);
  const remoteControlPlanes = controlPlanes.filter((c) => c.type === "remote");

  const [cpDialogOpen, setCpDialogOpen] = useState(false);
  const [editingCP, setEditingCP] = useState<ControlPlane | null>(null);

  const [workerDialogOpen, setWorkerDialogOpen] = useState(false);
  const [editingWorker, setEditingWorker] = useState<Worker | null>(null);
  const [workerDialogCpId, setWorkerDialogCpId] = useState<string>("");

  const handleAddCP = () => {
    setEditingCP(null);
    setCpDialogOpen(true);
  };

  const handleEditCP = (cp: ControlPlane) => {
    setEditingCP(cp);
    setCpDialogOpen(true);
  };

  const handleSaveCP = (data: { name: string; url: string; authToken?: string }) => {
    if (editingCP) {
      updateControlPlane(editingCP.id, data);
    } else {
      addControlPlane({
        name: data.name,
        type: "remote",
        url: data.url,
        authToken: data.authToken,
      });
    }
    setCpDialogOpen(false);
    setEditingCP(null);
  };

  const handleAddWorker = (controlPlaneId: string) => {
    setEditingWorker(null);
    setWorkerDialogCpId(controlPlaneId);
    setWorkerDialogOpen(true);
  };

  const handleEditWorker = (worker: Worker) => {
    setEditingWorker(worker);
    setWorkerDialogCpId(worker.controlPlaneId);
    setWorkerDialogOpen(true);
  };

  const handleSaveWorker = (data: { name: string; url: string; secret?: string }) => {
    if (editingWorker) {
      updateWorker(editingWorker.id, data);
    } else {
      addWorker({
        name: data.name,
        type: "remote",
        url: data.url,
        secret: data.secret,
        controlPlaneId: workerDialogCpId,
      });
    }
    setWorkerDialogOpen(false);
    setEditingWorker(null);
  };

  return (
    <>
      {/* Add control plane button */}
      <div className="mb-6 flex justify-end">
        <Button variant="outline" size="sm" onClick={handleAddCP}>
          <Plus className="size-3.5" />
          Add Control Plane
        </Button>
      </div>

      {/* Embedded control plane */}
      {embeddedCP && (
        <div className="mb-6 rounded-lg border border-border-card bg-card overflow-hidden">
          <div className="flex items-center justify-between gap-4 px-4 py-3">
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium text-foreground">Embedded Control Plane</div>
              <div className="mt-0.5 text-xs text-muted-foreground">{embeddedCP.url}</div>
            </div>
            <div className="flex items-center gap-2">
              <ServerStatusDot status="connected" />
              <span className="text-xs text-muted-foreground">running</span>
            </div>
          </div>
          <div className="border-t border-border-card px-4 py-2">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Workers
            </span>
          </div>
          {embeddedWorkers.map((worker) => (
            <EmbeddedWorkerItem
              key={worker.id}
              worker={worker}
              onEdit={handleEditWorker}
              onRemove={removeWorker}
            />
          ))}
          {embeddedWorkers.length === 0 && (
            <div className="px-4 py-3 border-t border-border-card text-sm text-muted-foreground">
              No workers connected.
            </div>
          )}
          <div className="px-4 py-3 border-t border-border-card">
            <Button variant="outline" size="sm" onClick={() => handleAddWorker(embeddedCP.id)}>
              <Plus className="size-3.5" />
              Add Worker
            </Button>
          </div>
        </div>
      )}

      {/* Remote control planes — each gets its own panel */}
      {remoteControlPlanes.map((cp) => {
        const cpWorkers = allWorkers.filter((w) => w.controlPlaneId === cp.id);
        return (
          <div
            key={cp.id}
            className="mb-6 rounded-lg border border-border-card bg-card overflow-hidden"
          >
            <div className="flex items-center justify-between gap-4 px-4 py-3">
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium text-foreground">{cp.name}</div>
                <div className="mt-0.5 text-xs text-muted-foreground">{cp.url}</div>
              </div>
              <div className="flex items-center gap-2">
                <ServerStatusDot status={cp.status} />
                <span className="text-xs text-muted-foreground capitalize">{cp.status}</span>
                <Switch
                  checked={activeControlPlaneId === cp.id}
                  onCheckedChange={(checked) => {
                    if (checked) setActiveControlPlane(cp.id);
                  }}
                />
                <Button variant="ghost" size="icon-xs" onClick={() => handleEditCP(cp)}>
                  <Pencil />
                </Button>
                <Button variant="ghost" size="icon-xs" onClick={() => removeControlPlane(cp.id)}>
                  <Trash2 />
                </Button>
              </div>
            </div>
            <div className="border-t border-border-card px-4 py-2">
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Workers
              </span>
            </div>
            {cpWorkers.map((worker) => (
              <WorkerWithAgents
                key={worker.id}
                worker={worker}
                nested
                onEdit={handleEditWorker}
                onRemove={removeWorker}
              />
            ))}
            {cpWorkers.length === 0 && (
              <div className="px-4 py-3 border-t border-border-card text-sm text-muted-foreground">
                No workers connected.
              </div>
            )}
            <div className="px-4 py-3 border-t border-border-card">
              <Button variant="outline" size="sm" onClick={() => handleAddWorker(cp.id)}>
                <Plus className="size-3.5" />
                Add Worker
              </Button>
            </div>
          </div>
        );
      })}

      <AddEditControlPlaneDialog
        open={cpDialogOpen}
        onOpenChange={setCpDialogOpen}
        controlPlane={editingCP}
        onSave={handleSaveCP}
      />

      <AddEditWorkerDialog
        open={workerDialogOpen}
        onOpenChange={setWorkerDialogOpen}
        worker={editingWorker}
        onSave={handleSaveWorker}
      />
    </>
  );
}

function useEmbeddedWorkerStatus() {
  const client = useClient(EmbeddedWorkerService);
  const [data, setData] = useState<WatchEmbeddedWorkerStatusResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    (async () => {
      try {
        for await (const msg of client.watchEmbeddedWorkerStatus(
          {},
          { signal: controller.signal },
        )) {
          setData(msg);
        }
      } catch (err) {
        if (!controller.signal.aborted) setError(err as Error);
      }
    })();
    return () => controller.abort();
  }, [client]);

  return { data, error };
}

function embeddedStatusToConnectionStatus(status: EmbeddedWorkerStatus): ConnectionStatus {
  switch (status) {
    case EmbeddedWorkerStatus.RUNNING:
      return "connected";
    case EmbeddedWorkerStatus.STARTING:
    case EmbeddedWorkerStatus.STOPPING:
      return "connecting";
    case EmbeddedWorkerStatus.ERRORED:
      return "error";
    default:
      return "disconnected";
  }
}

function embeddedStatusLabel(status: EmbeddedWorkerStatus): string {
  switch (status) {
    case EmbeddedWorkerStatus.RUNNING:
      return "running";
    case EmbeddedWorkerStatus.STARTING:
      return "starting";
    case EmbeddedWorkerStatus.STOPPING:
      return "stopping";
    case EmbeddedWorkerStatus.ERRORED:
      return "errored";
    case EmbeddedWorkerStatus.STOPPED:
      return "stopped";
    default:
      return "unknown";
  }
}

function EmbeddedWorkerControls() {
  const client = useClient(EmbeddedWorkerService);
  const { data } = useEmbeddedWorkerStatus();
  const status = data?.status ?? EmbeddedWorkerStatus.UNSPECIFIED;

  const isTransitioning =
    status === EmbeddedWorkerStatus.STARTING || status === EmbeddedWorkerStatus.STOPPING;
  const isRunning = status === EmbeddedWorkerStatus.RUNNING;

  const startMutation = useMutation({
    mutationFn: () => client.startEmbeddedWorker({}),
  });
  const stopMutation = useMutation({
    mutationFn: () => client.stopEmbeddedWorker({}),
  });
  const restartMutation = useMutation({
    mutationFn: () => client.restartEmbeddedWorker({}),
  });

  const anyMutating =
    startMutation.isPending || stopMutation.isPending || restartMutation.isPending;

  const handleToggle = (checked: boolean) => {
    if (checked) {
      startMutation.mutate();
    } else {
      stopMutation.mutate();
    }
  };

  return (
    <div className="flex items-center gap-2">
      {data?.error && status === EmbeddedWorkerStatus.ERRORED && (
        <span className="text-xs text-destructive truncate max-w-40" title={data.error}>
          {data.error}
        </span>
      )}
      <ServerStatusDot status={embeddedStatusToConnectionStatus(status)} />
      <span className="text-xs text-muted-foreground capitalize">
        {embeddedStatusLabel(status)}
      </span>
      {isRunning && (
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="outline" size="sm" disabled={anyMutating}>
              <RotateCcw />
              Restart
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Restart embedded worker?</AlertDialogTitle>
              <AlertDialogDescription>
                This will restart the embedded worker and stop all running workloads. Any
                in-progress tasks will be interrupted.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction variant="destructive" onClick={() => restartMutation.mutate()}>
                Restart
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
      <Switch
        checked={isRunning}
        onCheckedChange={handleToggle}
        disabled={isTransitioning || anyMutating}
      />
    </div>
  );
}

function EmbeddedWorkerItem({
  worker,
  onEdit,
  onRemove,
}: {
  worker: Worker;
  onEdit: (worker: Worker) => void;
  onRemove: (id: string) => void;
}) {
  const isEmbedded = worker.type === "local";
  const { data: streamData } = useEmbeddedWorkerStatus();
  const streamStatus = streamData?.status ?? EmbeddedWorkerStatus.UNSPECIFIED;

  // For the embedded worker, derive connection status from the streaming RPC.
  const effectiveStatus: ConnectionStatus = isEmbedded
    ? embeddedStatusToConnectionStatus(streamStatus)
    : worker.status;
  const isConnected = effectiveStatus === "connected";

  const client = useClient(SystemService);
  const cpClient = useClient(WorkerService);
  const { data, isLoading, isError } = useQuery({
    queryKey: ["agents", worker.id],
    queryFn: () => client.listAgents({ disableCache: false }),
    enabled: isConnected,
    refetchInterval: 5000,
    placeholderData: (prev) => prev,
  });
  const ping = useQuery({
    queryKey: ["ping", worker.id],
    queryFn: () => cpClient.pingWorker({ workerId: worker.id }),
    enabled: isConnected,
    refetchInterval: 5000,
  });

  const rttLabel = ping.isError ? "unreachable" : (ping.data?.duration ?? "--");

  return (
    <>
      <SettingsItem
        label={isEmbedded ? "Run embedded worker" : worker.name}
        description={
          isEmbedded ? (
            <span className={streamData?.listenAddr ? undefined : "invisible"}>
              {streamData?.listenAddr || "127.0.0.1:00000"}
            </span>
          ) : (
            worker.url
          )
        }
      >
        <div className="flex items-center gap-2">
          {isConnected && (
            <span className="text-[10px] font-mono text-muted-foreground tabular-nums">
              {rttLabel}
            </span>
          )}
          {isEmbedded ? (
            <EmbeddedWorkerControls />
          ) : (
            <>
              <ServerStatusDot status={effectiveStatus} />
              <span className="text-xs text-muted-foreground capitalize">{effectiveStatus}</span>
            </>
          )}
          {!isEmbedded && (
            <>
              <Button variant="ghost" size="icon-xs" onClick={() => onEdit(worker)}>
                <Pencil />
              </Button>
              <Button variant="ghost" size="icon-xs" onClick={() => onRemove(worker.id)}>
                <Trash2 />
              </Button>
            </>
          )}
        </div>
      </SettingsItem>
      {isConnected && (
        <div className="border-t border-border-card px-4 pt-2 pb-2">
          {isLoading && (
            <div className="py-1 text-xs text-muted-foreground">Discovering agents...</div>
          )}
          {isError && (
            <div className="py-1 text-xs text-muted-foreground">Could not load agents</div>
          )}
          {data?.agents.map((agent) => {
            const Icon = getHarnessIcon(agent.id) ?? Bot;
            return (
              <div key={agent.id} className="flex items-center gap-2 py-1">
                <Icon className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="text-xs text-foreground">{agent.name}</span>
                <span className="text-[11px] text-muted-foreground">{agent.version}</span>
                {!agent.enabled && (
                  <span className="ml-auto text-[10px] text-muted-foreground">disabled</span>
                )}
              </div>
            );
          })}
          {data && data.agents.length === 0 && (
            <div className="py-1 text-xs text-muted-foreground">No agents discovered</div>
          )}
        </div>
      )}
    </>
  );
}

function WorkerWithAgents({
  worker,
  nested,
  onEdit,
  onRemove,
}: {
  worker: Worker;
  nested?: boolean;
  onEdit: (worker: Worker) => void;
  onRemove: (id: string) => void;
}) {
  const client = useClient(SystemService);
  const cpClient = useClient(WorkerService);
  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ["agents", worker.id],
    queryFn: () => client.listAgents({ disableCache: false }),
    enabled: worker.status === "connected",
  });
  const ping = useQuery({
    queryKey: ["ping", worker.id],
    queryFn: () => cpClient.pingWorker({ workerId: worker.id }),
    enabled: worker.status === "connected",
    refetchInterval: 5000,
  });

  const rttLabel = ping.isError ? "unreachable" : (ping.data?.duration ?? "--");

  if (nested) {
    return (
      <>
        <SettingsItem label={worker.name} description={worker.url}>
          <div className="flex items-center gap-2">
            <span className="text-[10px] font-mono text-muted-foreground tabular-nums">
              {rttLabel}
            </span>
            <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-5 font-medium">
              {worker.type}
            </Badge>
            <ServerStatusDot status={worker.status} />
            <span className="text-xs text-muted-foreground capitalize">{worker.status}</span>
            {worker.status === "connected" && (
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => refetch()}
                disabled={isFetching}
              >
                <RefreshCw className={isFetching ? "animate-spin" : ""} />
              </Button>
            )}
            <Button variant="ghost" size="icon-xs" onClick={() => onEdit(worker)}>
              <Pencil />
            </Button>
            <Button variant="ghost" size="icon-xs" onClick={() => onRemove(worker.id)}>
              <Trash2 />
            </Button>
          </div>
        </SettingsItem>
        {worker.status === "connected" && (
          <div className="border-t border-border-card px-4 pt-2 pb-2">
            {isLoading && (
              <div className="py-1 text-xs text-muted-foreground">Discovering agents...</div>
            )}
            {isError && (
              <div className="py-1 text-xs text-muted-foreground">Could not load agents</div>
            )}
            {data?.agents.map((agent) => {
              const Icon = getHarnessIcon(agent.id) ?? Bot;
              return (
                <div key={agent.id} className="flex items-center gap-2 py-1">
                  <Icon className="size-3.5 shrink-0 text-muted-foreground" />
                  <span className="text-xs text-foreground">{agent.name}</span>
                  <span className="text-[11px] text-muted-foreground">{agent.version}</span>
                  {!agent.enabled && (
                    <span className="ml-auto text-[10px] text-muted-foreground">disabled</span>
                  )}
                </div>
              );
            })}
            {data && data.agents.length === 0 && (
              <div className="py-1 text-xs text-muted-foreground">No agents discovered</div>
            )}
          </div>
        )}
      </>
    );
  }

  return (
    <div className="rounded-lg border border-border-card bg-card overflow-hidden mb-3 last:mb-0">
      <div className="flex items-center justify-between gap-4 px-4 py-2.5">
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium text-foreground">{worker.name}</div>
          {worker.url && <div className="mt-0.5 text-xs text-muted-foreground">{worker.url}</div>}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-[10px] font-mono text-muted-foreground tabular-nums">
            {rttLabel}
          </span>
          <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-5 font-medium">
            {worker.type}
          </Badge>
          <ServerStatusDot status={worker.status} />
          <span className="text-xs text-muted-foreground capitalize">{worker.status}</span>
          {worker.status === "connected" && (
            <Button variant="ghost" size="icon-xs" onClick={() => refetch()} disabled={isFetching}>
              <RefreshCw className={isFetching ? "animate-spin" : ""} />
            </Button>
          )}
          <Button variant="ghost" size="icon-xs" onClick={() => onEdit(worker)}>
            <Pencil />
          </Button>
          <Button variant="ghost" size="icon-xs" onClick={() => onRemove(worker.id)}>
            <Trash2 />
          </Button>
        </div>
      </div>

      {worker.status === "connected" && (
        <div className="border-t border-border-card px-4 pt-2 pb-2">
          {isLoading && (
            <div className="py-1 text-xs text-muted-foreground">Discovering agents...</div>
          )}
          {isError && (
            <div className="py-1 text-xs text-muted-foreground">Could not load agents</div>
          )}
          {data?.agents.map((agent) => {
            const Icon = getHarnessIcon(agent.id) ?? Bot;
            return (
              <div key={agent.id} className="flex items-center gap-2 py-1">
                <Icon className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="text-xs text-foreground">{agent.name}</span>
                <span className="text-[11px] text-muted-foreground">{agent.version}</span>
                {!agent.enabled && (
                  <span className="ml-auto text-[10px] text-muted-foreground">disabled</span>
                )}
              </div>
            );
          })}
          {data && data.agents.length === 0 && (
            <div className="py-1 text-xs text-muted-foreground">No agents discovered</div>
          )}
        </div>
      )}
    </div>
  );
}

function AddEditControlPlaneDialog({
  open,
  onOpenChange,
  controlPlane,
  onSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  controlPlane: ControlPlane | null;
  onSave: (data: { name: string; url: string; authToken?: string }) => void;
}) {
  const [name, setName] = useState(controlPlane?.name ?? "");
  const [url, setUrl] = useState(controlPlane?.url ?? "");
  const [authToken, setAuthToken] = useState(controlPlane?.authToken ?? "");

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setName(controlPlane?.name ?? "");
      setUrl(controlPlane?.url ?? "");
      setAuthToken(controlPlane?.authToken ?? "");
    }
    onOpenChange(nextOpen);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !url.trim()) return;
    onSave({
      name: name.trim(),
      url: url.trim(),
      authToken: authToken.trim() || undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{controlPlane ? "Edit Control Plane" : "Add Control Plane"}</DialogTitle>
          <DialogDescription>
            {controlPlane
              ? "Update the remote control plane configuration."
              : "Add a new remote control plane."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cp-name">Name</Label>
            <Input
              id="cp-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. staging-cp"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cp-url">URL</Label>
            <Input
              id="cp-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="e.g. https://cp.example.com:8420"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cp-token">Auth Token (optional)</Label>
            <Input
              id="cp-token"
              type="password"
              value={authToken}
              onChange={(e) => setAuthToken(e.target.value)}
              placeholder="Bearer token for authentication"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!name.trim() || !url.trim()}>
              {controlPlane ? "Save" : "Add Control Plane"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

const K8S_NAME_RE = /^[a-z]([-a-z0-9]*[a-z0-9])?$/;

function slugifyWorkerId(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-/, "")
    .replace(/-$/, "")
    .slice(0, 63);
}

function AddEditWorkerDialog({
  open,
  onOpenChange,
  worker,
  onSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  worker: Worker | null;
  onSave: (data: { id?: string; name: string; url: string; secret?: string }) => void;
}) {
  const [name, setName] = useState(worker?.name ?? "");
  const [workerId, setWorkerId] = useState("");
  const [idManuallyEdited, setIdManuallyEdited] = useState(false);
  const [url, setUrl] = useState(worker?.url ?? "");
  const [secret, setSecret] = useState(worker?.secret ?? "");
  const isEditing = !!worker;

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setName(worker?.name ?? "");
      setWorkerId("");
      setIdManuallyEdited(false);
      setUrl(worker?.url ?? "");
      setSecret(worker?.secret ?? "");
    }
    onOpenChange(nextOpen);
  };

  const handleNameChange = (value: string) => {
    setName(value);
    if (!isEditing && !idManuallyEdited) {
      setWorkerId(slugifyWorkerId(value));
    }
  };

  const handleIdChange = (value: string) => {
    setIdManuallyEdited(true);
    setWorkerId(value);
  };

  const idValid = isEditing || workerId === "" || K8S_NAME_RE.test(workerId);
  const canSubmit =
    name.trim() !== "" &&
    url.trim() !== "" &&
    (isEditing || (workerId !== "" && K8S_NAME_RE.test(workerId)));

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    onSave({
      id: isEditing ? undefined : workerId,
      name: name.trim(),
      url: url.trim(),
      secret: secret.trim() || undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{worker ? "Edit Worker" : "Add Worker"}</DialogTitle>
          <DialogDescription>
            {worker
              ? "Update the worker configuration."
              : "Connect a new worker to this control plane."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="worker-name">Name</Label>
            <Input
              id="worker-name"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="e.g. My Worker"
            />
          </div>
          {!isEditing && (
            <div className="space-y-2">
              <Label htmlFor="worker-id">ID</Label>
              <Input
                id="worker-id"
                value={workerId}
                onChange={(e) => handleIdChange(e.target.value)}
                placeholder="e.g. my-worker"
                className={!idValid ? "border-destructive" : undefined}
              />
              {!idValid && (
                <p className="text-xs text-destructive">
                  Must be lowercase alphanumeric with hyphens, starting with a letter (max 63
                  chars).
                </p>
              )}
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="worker-url">URL</Label>
            <Input
              id="worker-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="e.g. http://localhost:8081"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="worker-secret">Secret</Label>
            <Input
              id="worker-secret"
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              placeholder="Worker authentication secret"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!canSubmit}>
              {worker ? "Save" : "Add Worker"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
