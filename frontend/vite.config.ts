import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

declare const process: { env: Record<string, string | undefined> };

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: "/ui/",
  build: {
    outDir: "../dist",
    emptyOutDir: true,
  },
  server: {
    port: Number(process.env.VITE_PORT ?? process.env.PORT ?? 5173),
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://localhost:7330",
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
