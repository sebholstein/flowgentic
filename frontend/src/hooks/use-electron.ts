export function useIsElectron(): boolean {
  return typeof window !== "undefined" && !!window.electronAPI;
}

export function useIsMacOS(): boolean {
  return typeof window !== "undefined" && window.electronAPI?.platform === "darwin";
}
