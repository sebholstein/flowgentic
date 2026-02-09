import { useMemo, type ReactNode } from "react";
import { useInfrastructureStore, selectActiveControlPlane } from "@/stores/serverStore";
import { TransportContext, createTransport } from "@/lib/connect";

export function ConnectProvider({ children }: { children: ReactNode }) {
  const activeCP = useInfrastructureStore(selectActiveControlPlane);
  const transport = useMemo(() => createTransport(activeCP.url), [activeCP.url]);

  return <TransportContext value={transport}>{children}</TransportContext>;
}
