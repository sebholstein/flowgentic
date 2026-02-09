import { create } from "zustand";
import type { ControlPlane, Worker, ConnectionStatus } from "@/types/server";
import { controlPlanes as mockControlPlanes, workers as mockWorkers } from "@/data/mockServerData";

interface InfrastructureState {
  controlPlanes: ControlPlane[];
  activeControlPlaneId: string;
  workers: Worker[];
}

interface InfrastructureActions {
  addControlPlane: (cp: Omit<ControlPlane, "id" | "status">) => void;
  removeControlPlane: (id: string) => void;
  updateControlPlane: (id: string, updates: Partial<ControlPlane>) => void;
  setActiveControlPlane: (id: string) => void;
  setControlPlaneStatus: (id: string, status: ConnectionStatus) => void;
  addWorker: (worker: Omit<Worker, "id" | "status">) => void;
  removeWorker: (id: string) => void;
  updateWorker: (id: string, updates: Partial<Worker>) => void;
  setWorkers: (workers: Worker[]) => void;
  setWorkerStatus: (id: string, status: ConnectionStatus) => void;
}

type InfrastructureStore = InfrastructureState & InfrastructureActions;

const initialState: InfrastructureState = {
  controlPlanes: mockControlPlanes,
  activeControlPlaneId: "cp-embedded",
  workers: mockWorkers,
};

export const useInfrastructureStore = create<InfrastructureStore>((set) => ({
  ...initialState,

  addControlPlane: (cpData) => {
    const newCP: ControlPlane = {
      ...cpData,
      id: `cp-remote-${Date.now()}`,
      status: "connecting",
    };
    set((state) => ({
      controlPlanes: [...state.controlPlanes, newCP],
    }));
  },

  removeControlPlane: (id) => {
    set((state) => {
      const cp = state.controlPlanes.find((c) => c.id === id);
      if (!cp || cp.type === "embedded") return state;
      return {
        controlPlanes: state.controlPlanes.filter((c) => c.id !== id),
        activeControlPlaneId:
          state.activeControlPlaneId === id ? "cp-embedded" : state.activeControlPlaneId,
      };
    });
  },

  updateControlPlane: (id, updates) => {
    set((state) => ({
      controlPlanes: state.controlPlanes.map((c) => (c.id === id ? { ...c, ...updates } : c)),
    }));
  },

  setActiveControlPlane: (id) => {
    set({ activeControlPlaneId: id });
  },

  setControlPlaneStatus: (id, status) => {
    set((state) => ({
      controlPlanes: state.controlPlanes.map((c) => (c.id === id ? { ...c, status } : c)),
    }));
  },

  addWorker: (workerData) => {
    const newWorker: Worker = {
      ...workerData,
      id: `worker-${Date.now()}`,
      status: "connecting",
    };
    set((state) => ({
      workers: [...state.workers, newWorker],
    }));
  },

  removeWorker: (id) => {
    set((state) => ({
      workers: state.workers.filter((w) => w.id !== id),
    }));
  },

  updateWorker: (id, updates) => {
    set((state) => ({
      workers: state.workers.map((w) => (w.id === id ? { ...w, ...updates } : w)),
    }));
  },

  setWorkers: (workers) => {
    set({ workers });
  },

  setWorkerStatus: (id, status) => {
    set((state) => ({
      workers: state.workers.map((w) => (w.id === id ? { ...w, status } : w)),
    }));
  },
}));

// Selectors
export const selectActiveControlPlane = (state: InfrastructureState) =>
  state.controlPlanes.find((c) => c.id === state.activeControlPlaneId) ?? state.controlPlanes[0];

export const selectRemoteControlPlanes = (state: InfrastructureState) =>
  state.controlPlanes.filter((c) => c.type === "remote");

export const selectActiveWorkers = (state: InfrastructureState) => {
  const activeCP = selectActiveControlPlane(state);
  return state.workers.filter((w) => w.controlPlaneId === activeCP.id);
};

export const selectControlPlaneById = (state: InfrastructureState, id: string) =>
  state.controlPlanes.find((c) => c.id === id);

// Backward-compat re-exports
export const useServerStore = useInfrastructureStore;
export const selectDefaultServer = selectActiveControlPlane;
