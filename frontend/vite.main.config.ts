import { defineConfig } from "vite";

export default defineConfig({
  build: {
    lib: {
      entry: "src/electron/main.ts",
      formats: ["cjs"],
      fileName: () => "main.cjs",
    },
    rollupOptions: {
      external: ["electron", "electron-squirrel-startup", "node:path"],
    },
  },
});
