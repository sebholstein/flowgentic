import { queryOptions } from "@tanstack/react-query";
import type { Client } from "@connectrpc/connect";
import type { TaskService } from "@/proto/gen/controlplane/v1/task_service_pb";

export function tasksQueryOptions(
  client: Client<typeof TaskService>,
  threadId: string,
) {
  return queryOptions({
    queryKey: ["tasks", threadId],
    queryFn: () => client.listTasks({ threadId }),
    enabled: !!threadId,
    refetchInterval: 3000,
  });
}
