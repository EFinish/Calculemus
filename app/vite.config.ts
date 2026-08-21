import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  // The sharing server (server/) owns /api; in production it also serves the
  // built app, so the proxy exists only for dev and e2e.
  server: {
    proxy: { "/api": "http://localhost:8737" },
  },
});
