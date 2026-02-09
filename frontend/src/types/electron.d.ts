export interface ElectronAPI {
  platform: NodeJS.Platform;
  send: (channel: string, data: unknown) => void;
  invoke: (channel: string, data: unknown) => Promise<unknown>;
}

declare global {
  interface Window {
    electronAPI?: ElectronAPI;
  }
}
