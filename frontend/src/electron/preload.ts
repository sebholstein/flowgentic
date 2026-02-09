import { contextBridge, ipcRenderer } from "electron";

contextBridge.exposeInMainWorld("electronAPI", {
  platform: process.platform,
  send: (channel: string, data: unknown) => {
    const validChannels = ["toMain"];
    if (validChannels.includes(channel)) ipcRenderer.send(channel, data);
  },
  invoke: async (channel: string, data: unknown) => {
    const validChannels = ["dialog:openFile", "app:getVersion"];
    if (validChannels.includes(channel)) return ipcRenderer.invoke(channel, data);
    return null;
  },
});
