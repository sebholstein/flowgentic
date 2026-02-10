import { queryOptions } from "@tanstack/react-query";
import type { Client } from "@connectrpc/connect";
import type { WorkerService } from "@/proto/gen/controlplane/v1/worker_service_pb";

export function workersQueryOptions(client: Client<typeof WorkerService>) {
  return queryOptions({
    queryKey: ["workers"],
    queryFn: () => client.listWorkers({}),
  });
}
