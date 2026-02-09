import type { ControlPlane, Worker } from "@/types/server";
import { CONTROL_PLANE_URL } from "@/lib/config";

export const controlPlanes: ControlPlane[] = [
  {
    id: "cp-embedded",
    name: "Local",
    type: "embedded",
    url: CONTROL_PLANE_URL,
    status: "connected",
    version: "0.1.0",
  },
];

export const workers: Worker[] = [
  {
    id: "local",
    name: "Local Worker",
    type: "local",
    url: "localhost:8081",
    status: "connected",
    controlPlaneId: "cp-embedded",
  },
];
