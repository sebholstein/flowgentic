import { defineConfig } from "vite";
import { devtools } from "@tanstack/devtools-vite";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import viteReact from "@vitejs/plugin-react";
import viteTsConfigPaths from "vite-tsconfig-paths";
import { fileURLToPath, URL } from "url";

import tailwindcss from "@tailwindcss/vite";

const config = defineConfig({
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  plugins: [
    devtools(),
    // this is the plugin that enables path aliases
    viteTsConfigPaths({
      projects: ["./tsconfig.json"],
    }),
    tailwindcss(),
    TanStackRouterVite(),
    viteReact(),
  ],
});

export default config;
