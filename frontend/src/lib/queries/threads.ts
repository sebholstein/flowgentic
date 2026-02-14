import { queryOptions } from "@tanstack/react-query";
import type { Client } from "@connectrpc/connect";
import type { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";

export function threadsQueryOptions(
  client: Client<typeof ThreadService>,
  projectId: string,
) {
  return queryOptions({
    queryKey: ["threads", projectId],
    queryFn: () => client.listThreads({ projectId }),
    enabled: !!projectId,
    refetchInterval: 2000,
  });
}
