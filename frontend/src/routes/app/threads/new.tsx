import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useClient } from "@/lib/connect";
import { ProjectService } from "@/proto/gen/controlplane/v1/project_service_pb";
import { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";
import { projectsQueryOptions } from "@/lib/queries/projects";
import type { Project } from "@/types/project";

type SearchParams = {
  projectId?: string;
  mode?: string;
};

export const Route = createFileRoute("/app/threads/new")({
  component: NewThreadPage,
  validateSearch: (search: Record<string, unknown>): SearchParams => ({
    projectId: typeof search.projectId === "string" ? search.projectId : undefined,
    mode: typeof search.mode === "string" ? search.mode : undefined,
  }),
});

function NewThreadPage() {
  const { projectId, mode } = Route.useSearch();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const projectClient = useClient(ProjectService);
  const threadClient = useClient(ThreadService);
  const { data: projectsData } = useQuery(projectsQueryOptions(projectClient));

  const projects = useMemo<Project[]>(
    () =>
      (projectsData?.projects ?? []).map((p) => ({
        id: p.id,
        name: p.name,
        defaultPlannerAgent: p.defaultPlannerAgent,
        defaultPlannerModel: p.defaultPlannerModel,
        embeddedWorkerPath: p.embeddedWorkerPath,
        workerPaths: p.workerPaths,
        sortIndex: p.sortIndex,
      })),
    [projectsData],
  );

  const currentProject = projects.find((p) => p.id === projectId) ?? projects[0];

  const createThreadMutation = useMutation({
    mutationFn: () =>
      threadClient.createThread({
        projectId: currentProject?.id ?? "",
        mode: mode ?? "plan",
      }),
    onSuccess: (resp) => {
      const id = resp.thread?.id ?? "";
      queryClient.invalidateQueries({ queryKey: ["threads"] });
      navigate({
        to: "/app/threads/$threadId",
        params: { threadId: id },
        replace: true,
      });
    },
  });

  // Immediately create the thread on mount
  if (!createThreadMutation.isPending && !createThreadMutation.isSuccess && !createThreadMutation.isError) {
    createThreadMutation.mutate();
  }

  return (
    <div className="flex h-full items-center justify-center text-muted-foreground">
      <span className="text-sm">Creating thread…</span>
    </div>
  );
}
