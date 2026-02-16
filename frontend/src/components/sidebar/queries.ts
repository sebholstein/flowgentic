import { useMemo } from "react";
import { useQuery, useQueries, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useClient } from "@/lib/connect";
import { ProjectService } from "@/proto/gen/controlplane/v1/project_service_pb";
import { WorkerService } from "@/proto/gen/controlplane/v1/worker_service_pb";
import { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";
import { projectsQueryOptions } from "@/lib/queries/projects";
import { workersQueryOptions } from "@/lib/queries/workers";
import { threadsQueryOptions } from "@/lib/queries/threads";
import type { ThreadConfig } from "@/proto/gen/controlplane/v1/thread_service_pb";
import type { Project } from "@/types/project";

export function useSidebarData() {
  const projectClient = useClient(ProjectService);
  const workerClient = useClient(WorkerService);
  const threadClient = useClient(ThreadService);

  const { data: projectsData } = useQuery(projectsQueryOptions(projectClient));
  const { data: workersData } = useQuery(workersQueryOptions(workerClient));

  const fetchedProjects = useMemo<Project[]>(
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

  const threadQueries = useQueries({
    queries: fetchedProjects.map((p) => threadsQueryOptions(threadClient, p.id)),
  });

  const backendThreads = useMemo<ThreadConfig[]>(() => {
    const result: ThreadConfig[] = [];
    for (const q of threadQueries) {
      for (const t of q.data?.threads ?? []) {
        result.push(t);
      }
    }
    return result;
  }, [threadQueries]);

  return { fetchedProjects, backendThreads, workersData, projectClient, threadClient };
}

export function useArchiveThread() {
  const threadClient = useClient(ThreadService);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { id: string; archived: boolean }) =>
      threadClient.archiveThread({ id: data.id, archived: data.archived }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["threads"] });
    },
  });
}

export function useCreateThread() {
  const threadClient = useClient(ThreadService);
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (data: { projectId: string }) =>
      threadClient.createThread({ projectId: data.projectId }),
    onSuccess: (resp) => {
      const id = resp.thread?.id ?? "";
      queryClient.invalidateQueries({ queryKey: ["threads"] });
      navigate({ to: "/app/threads/$threadId", params: { threadId: id } });
    },
  });
}

export function useCreateProject(callbacks: {
  onExpandProject: (id: string) => void;
  onCloseDialog: () => void;
}) {
  const projectClient = useClient(ProjectService);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: {
      id: string;
      name: string;
      defaultPlannerAgent: string;
      defaultPlannerModel: string;
      embeddedWorkerPath: string;
      workerPaths: Record<string, string>;
    }) =>
      projectClient.createProject({
        id: data.id,
        name: data.name,
        defaultPlannerAgent: data.defaultPlannerAgent,
        defaultPlannerModel: data.defaultPlannerModel,
        embeddedWorkerPath: data.embeddedWorkerPath,
        workerPaths: data.workerPaths,
      }),
    onSuccess: (resp) => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
      if (resp.project) {
        callbacks.onExpandProject(resp.project.id);
      }
      callbacks.onCloseDialog();
    },
  });
}

export function useReorderProjects() {
  const projectClient = useClient(ProjectService);
  const queryClient = useQueryClient();

  return {
    reorder: (entries: { id: string; sortIndex: number }[]) =>
      projectClient.reorderProjects({ entries }).then(() => {
        queryClient.invalidateQueries({ queryKey: ["projects"] });
      }),
  };
}
