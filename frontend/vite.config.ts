import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";
import { defineConfig } from "vite";

declare const process: { env: Record<string, string | undefined> };

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: "/ui/",
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
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
