import { defineConfig } from "vite";

export default defineConfig({
  clearScreen: false,
  base: "./",
  server: {
    port: 5179,
    strictPort: true,
  },
  envPrefix: ["VITE_", "TAURI_"],
});
