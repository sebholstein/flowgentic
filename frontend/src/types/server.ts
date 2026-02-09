export type ConnectionStatus = "connecting" | "connected" | "disconnected" | "error";

export interface ControlPlane {
  id: string;
  name: string;
  type: "embedded" | "remote";
  url: string;
  status: ConnectionStatus;
  authToken?: string;
  lastConnectedAt?: string;
  version?: string;
}

export interface Worker {
  id: string;
  name: string;
  type: "local" | "remote";
  url?: string;
  secret?: string;
  status: ConnectionStatus;
  controlPlaneId: string;
  capabilities?: string[];
  lastActiveAt?: string;
}
