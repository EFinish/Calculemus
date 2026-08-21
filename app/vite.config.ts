import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  // The sharing server (server/) owns /api; in production it also serves the
  // built app, so the proxy exists only for dev and e2e.
  server: {
    proxy: { "/api": "http://localhost:8737" },
    // The Try-me gallery imports ../examples/*.json (canonical copies shared
    // with the CLI); dev serving needs the parent dir allowed.
    fs: { allow: [".."] },
  },
});
