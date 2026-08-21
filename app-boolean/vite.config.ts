import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  // Relative base: this frozen edition is mounted at /boolean/ by the server;
  // relative asset paths work there and at dev root alike. Share links use a
  // query param (?u=) instead of a path route for the same reason.
  base: "./",
  plugins: [vue()],
  // The sharing server (server/) owns /api; in production it also serves the
  // built app, so the proxy exists only for dev and e2e.
  server: {
    proxy: { "/api": "http://localhost:8737" },
  },
});
