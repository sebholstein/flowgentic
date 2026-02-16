import { queryOptions } from "@tanstack/react-query";
import type { Client } from "@connectrpc/connect";
import type { SessionService } from "@/proto/gen/controlplane/v1/session_service_pb";

export function sessionsQueryOptions(client: Client<typeof SessionService>, threadId: string) {
  return queryOptions({
    queryKey: ["sessions", threadId],
    queryFn: () => client.listSessions({ threadId }),
    enabled: !!threadId,
    refetchInterval: 5000,
  });
}
