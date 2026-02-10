import { queryOptions } from "@tanstack/react-query";
import type { Client } from "@connectrpc/connect";
import type { ProjectService } from "@/proto/gen/controlplane/v1/project_service_pb";

export function projectsQueryOptions(client: Client<typeof ProjectService>) {
  return queryOptions({
    queryKey: ["projects"],
    queryFn: () => client.listProjects({}),
  });
}
