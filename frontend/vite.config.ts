import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: "/ui/",
  build: {
    outDir: "../dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:7330",
        changeOrigin: true,
        secure: false,
      },
      "/oauth": {
        target: "http://localhost:7330",
        changeOrigin: true,
        secure: false,
      },
      "/auth": {
        target: "http://localhost:7330",
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
