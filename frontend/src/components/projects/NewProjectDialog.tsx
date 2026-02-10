import { useRef } from "react";
import { useForm, useFormContext, Controller } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Form } from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { WorkerConfig } from "@/proto/gen/controlplane/v1/worker_service_pb";
import type { CreateProjectRequest } from "@/proto/gen/controlplane/v1/project_service_pb";

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

function validateResourceName(value: string) {
  if (!value) return "Required";
  if (value.length > MAX_NAME_LEN) return `Max ${MAX_NAME_LEN} characters`;
  if (!K8S_NAME_RE.test(value))
    return "Must be lowercase alphanumeric with hyphens, starting with a letter.";
  return true;
}

type ProjectFormValues = Pick<
  CreateProjectRequest,
  "id" | "name" | "defaultPlannerAgent" | "defaultPlannerModel" | "embeddedWorkerPath" | "workerPaths"
> & {
  defaultWorkerId: string;
};

function NameAndIdFields() {
  const { register, setValue, formState: { errors } } = useFormContext<ProjectFormValues>();
  const idManuallyEdited = useRef(false);

  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="project-name">Name</Label>
        <Input
          id="project-name"
          {...register("name", { required: "Required" })}
          onChange={(e) => {
            const value = e.target.value;
            setValue("name", value, { shouldValidate: true });
            if (!idManuallyEdited.current) {
              setValue("id", slugify(value), { shouldValidate: true });
            }
          }}
          placeholder="e.g. My Project"
          autoFocus
        />
        {errors.name && (
          <p className="text-xs text-destructive">{errors.name.message}</p>
        )}
      </div>
      <div className="space-y-2">
        <Label htmlFor="project-id">ID</Label>
        <Input
          id="project-id"
          {...register("id", { validate: validateResourceName })}
          onChange={(e) => {
            idManuallyEdited.current = true;
            setValue("id", e.target.value, { shouldValidate: true });
          }}
          placeholder="e.g. my-project"
          className={errors.id ? "border-destructive" : undefined}
        />
        {errors.id && (
          <p className="text-xs text-destructive">{errors.id.message}</p>
        )}
      </div>
    </>
  );
}

function WorkerSelectField({ workers }: { workers: WorkerConfig[] }) {
  const { control } = useFormContext<ProjectFormValues>();

  if (workers.length === 0) return null;

  return (
    <div className="space-y-2">
      <Label>Default Worker</Label>
      <Controller
        control={control}
        name="defaultWorkerId"
        render={({ field }) => (
          <Select value={field.value} onValueChange={field.onChange}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Select a worker…" />
            </SelectTrigger>
            <SelectContent>
              {workers.map((worker) => (
                <SelectItem key={worker.id} value={worker.id}>
                  {worker.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      />
    </div>
  );
}

function WorkerPathFields({ workers }: { workers: WorkerConfig[] }) {
  const { register } = useFormContext<ProjectFormValues>();

  if (workers.length === 0) return null;

  return (
    <div className="space-y-2">
      <Label>Worker Paths</Label>
      {workers.map((worker) => (
        <div key={worker.id} className="space-y-1">
          <Label htmlFor={`worker-path-${worker.id}`} className="text-xs text-muted-foreground">
            {worker.name}
          </Label>
          <Input
            id={`worker-path-${worker.id}`}
            {...register(`workerPaths.${worker.id}`)}
            placeholder="/path/to/worker"
          />
        </div>
      ))}
    </div>
  );
}

export function NewProjectDialog({
  open,
  onOpenChange,
  onSave,
  workers,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (data: ProjectFormValues) => void;
  workers: WorkerConfig[];
}) {
  const form = useForm<ProjectFormValues>({
    defaultValues: {
      id: "",
      name: "",
      defaultPlannerAgent: "",
      defaultPlannerModel: "",
      embeddedWorkerPath: "",
      workerPaths: {},
      defaultWorkerId: "",
    },
  });

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      form.reset();
    }
    onOpenChange(nextOpen);
  };

  const handleSubmit = (data: ProjectFormValues) => {
    // Strip empty worker paths
    const workerPaths: Record<string, string> = {};
    for (const [key, value] of Object.entries(data.workerPaths)) {
      const trimmed = value.trim();
      if (trimmed) {
        workerPaths[key] = trimmed;
      }
    }

    onSave({
      ...data,
      name: data.name.trim(),
      defaultPlannerAgent: data.defaultPlannerAgent.trim(),
      defaultPlannerModel: data.defaultPlannerModel.trim(),
      embeddedWorkerPath: data.embeddedWorkerPath.trim(),
      workerPaths,
    });
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>New Project</DialogTitle>
          <DialogDescription>Create a new project for organizing threads and tasks.</DialogDescription>
        </DialogHeader>
        <Form form={form} onSubmit={handleSubmit} className="space-y-4">
          <NameAndIdFields />
          <div className="space-y-2">
            <Label htmlFor="project-planner-agent">Default Planner Agent</Label>
            <Input
              id="project-planner-agent"
              {...form.register("defaultPlannerAgent")}
              placeholder="Optional"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="project-planner-model">Default Planner Model</Label>
            <Input
              id="project-planner-model"
              {...form.register("defaultPlannerModel")}
              placeholder="Optional"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="project-embedded-worker-path">Embedded Worker Path</Label>
            <Input
              id="project-embedded-worker-path"
              {...form.register("embeddedWorkerPath")}
              placeholder="Optional — absolute file path"
            />
          </div>

          <WorkerSelectField workers={workers} />
          <WorkerPathFields workers={workers} />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit">
              Create Project
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
