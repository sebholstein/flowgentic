import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
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

const K8S_NAME_RE = /^[a-z]([-a-z0-9]*[a-z0-9])?$/;
const MAX_NAME_LEN = 63;

function slugify(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-/, "")
    .replace(/-$/, "")
    .slice(0, MAX_NAME_LEN);
}

function isValidResourceName(name: string): boolean {
  return name.length > 0 && name.length <= MAX_NAME_LEN && K8S_NAME_RE.test(name);
}

interface WorkerPathEntry {
  key: string;
  workerName: string;
  path: string;
}

export function NewProjectDialog({
  open,
  onOpenChange,
  onSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (data: {
    id: string;
    name: string;
    defaultPlannerAgent: string;
    defaultPlannerModel: string;
    embeddedWorkerPath: string;
    workerPaths: Record<string, string>;
  }) => void;
}) {
  const [name, setName] = useState("");
  const [id, setId] = useState("");
  const [idManuallyEdited, setIdManuallyEdited] = useState(false);
  const [defaultPlannerAgent, setDefaultPlannerAgent] = useState("");
  const [defaultPlannerModel, setDefaultPlannerModel] = useState("");
  const [embeddedWorkerPath, setEmbeddedWorkerPath] = useState("");
  const [workerPaths, setWorkerPaths] = useState<WorkerPathEntry[]>([]);
  let nextKey = 0;

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setName("");
      setId("");
      setIdManuallyEdited(false);
      setDefaultPlannerAgent("");
      setDefaultPlannerModel("");
      setEmbeddedWorkerPath("");
      setWorkerPaths([]);
    }
    onOpenChange(nextOpen);
  };

  const handleNameChange = (value: string) => {
    setName(value);
    if (!idManuallyEdited) {
      setId(slugify(value));
    }
  };

  const handleIdChange = (value: string) => {
    setIdManuallyEdited(true);
    setId(value);
  };

  const addWorkerPath = () => {
    setWorkerPaths((prev) => [...prev, { key: `wp-${Date.now()}-${nextKey++}`, workerName: "", path: "" }]);
  };

  const removeWorkerPath = (key: string) => {
    setWorkerPaths((prev) => prev.filter((wp) => wp.key !== key));
  };

  const updateWorkerPath = (key: string, field: "workerName" | "path", value: string) => {
    setWorkerPaths((prev) =>
      prev.map((wp) => (wp.key === key ? { ...wp, [field]: value } : wp)),
    );
  };

  const idValid = id === "" || isValidResourceName(id);
  const workerNamesValid = workerPaths.every(
    (wp) => wp.workerName === "" || isValidResourceName(wp.workerName),
  );
  const canSubmit = name.trim() !== "" && id !== "" && isValidResourceName(id) && workerNamesValid;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;

    const wp: Record<string, string> = {};
    for (const entry of workerPaths) {
      if (entry.workerName && entry.path) {
        wp[entry.workerName] = entry.path;
      }
    }

    onSave({
      id,
      name: name.trim(),
      defaultPlannerAgent: defaultPlannerAgent.trim(),
      defaultPlannerModel: defaultPlannerModel.trim(),
      embeddedWorkerPath: embeddedWorkerPath.trim(),
      workerPaths: wp,
    });
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>New Project</DialogTitle>
          <DialogDescription>Create a new project for organizing threads and tasks.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="project-name">Name</Label>
            <Input
              id="project-name"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="e.g. My Project"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="project-id">ID</Label>
            <Input
              id="project-id"
              value={id}
              onChange={(e) => handleIdChange(e.target.value)}
              placeholder="e.g. my-project"
              className={!idValid ? "border-destructive" : undefined}
            />
            {!idValid && (
              <p className="text-xs text-destructive">
                Must be lowercase alphanumeric with hyphens, starting with a letter (max 63 chars).
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="project-planner-agent">Default Planner Agent</Label>
            <Input
              id="project-planner-agent"
              value={defaultPlannerAgent}
              onChange={(e) => setDefaultPlannerAgent(e.target.value)}
              placeholder="Optional"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="project-planner-model">Default Planner Model</Label>
            <Input
              id="project-planner-model"
              value={defaultPlannerModel}
              onChange={(e) => setDefaultPlannerModel(e.target.value)}
              placeholder="Optional"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="project-embedded-worker-path">Embedded Worker Path</Label>
            <Input
              id="project-embedded-worker-path"
              value={embeddedWorkerPath}
              onChange={(e) => setEmbeddedWorkerPath(e.target.value)}
              placeholder="Optional — absolute file path"
            />
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Worker Paths</Label>
              <Button type="button" variant="outline" size="sm" onClick={addWorkerPath}>
                <Plus className="size-3.5" />
                Add
              </Button>
            </div>
            {workerPaths.map((wp) => {
              const nameValid = wp.workerName === "" || isValidResourceName(wp.workerName);
              return (
                <div key={wp.key} className="flex items-start gap-2">
                  <div className="flex-1 space-y-1">
                    <Input
                      value={wp.workerName}
                      onChange={(e) => updateWorkerPath(wp.key, "workerName", e.target.value)}
                      placeholder="worker-name"
                      className={!nameValid ? "border-destructive" : undefined}
                    />
                    {!nameValid && (
                      <p className="text-xs text-destructive">Invalid resource name.</p>
                    )}
                  </div>
                  <div className="flex-1">
                    <Input
                      value={wp.path}
                      onChange={(e) => updateWorkerPath(wp.key, "path", e.target.value)}
                      placeholder="/path/to/worker"
                    />
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    onClick={() => removeWorkerPath(wp.key)}
                    className="mt-1.5"
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>
              );
            })}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!canSubmit}>
              Create Project
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
