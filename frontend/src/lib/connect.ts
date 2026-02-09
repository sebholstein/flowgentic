import { useMemo, useContext, createContext } from "react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import type { Transport, Client } from "@connectrpc/connect";
import type { DescService } from "@bufbuild/protobuf";

export const TransportContext = createContext<Transport | null>(null);

export function useTransport(): Transport {
  const transport = useContext(TransportContext);
  if (!transport) {
    throw new Error("useTransport must be used within a ConnectProvider");
  }
  return transport;
}

export function useClient<T extends DescService>(service: T): Client<T> {
  const transport = useTransport();
  return useMemo(() => createClient(service, transport), [service, transport]);
}

export function createTransport(baseUrl: string) {
  return createConnectTransport({ baseUrl });
}
