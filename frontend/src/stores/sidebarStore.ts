import { create } from "zustand";

interface SidebarState {
  visible: boolean;
  width: number;
}

interface SidebarActions {
  toggle: () => void;
  show: () => void;
  hide: () => void;
  setWidth: (width: number) => void;
}

type SidebarStore = SidebarState & SidebarActions;

const clampWidth = (w: number) => Math.min(600, Math.max(200, w));

export const useSidebarStore = create<SidebarStore>((set) => ({
  visible: true,
  width: 320,

  toggle: () => set((s) => ({ visible: !s.visible })),
  show: () => set({ visible: true }),
  hide: () => set({ visible: false }),
  setWidth: (width) => set({ width: clampWidth(width) }),
}));
