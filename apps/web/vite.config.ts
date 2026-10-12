import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// The browser only ever talks to this server; /api/* is proxied to the Go API,
// so there is one origin, no CORS, and the auth cookie just works.
const apiUrl = process.env.API_URL ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  server: {
    port: 3000,
    strictPort: true,
    proxy: { "/api": apiUrl },
  },
  build: {
    // SurveyJS (with packages/clinical) is one ~1.5 MB chunk. It loads only on the pages that use it.
    chunkSizeWarningLimit: 1600,
  },
  preview: {
    port: 3000,
    strictPort: true,
    proxy: { "/api": apiUrl },
  },
  test: {
    environment: "node",
  },
});
