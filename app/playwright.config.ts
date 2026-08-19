import { defineConfig, devices } from "@playwright/test";

// Firefox on purpose: the e2e dogfood runs without any browser-extension
// pairing, so it works headless and in CI.
export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  use: { baseURL: "http://localhost:5199" },
  projects: [{ name: "firefox", use: { ...devices["Desktop Firefox"] } }],
  webServer: {
    // Assumes `make wasm` has populated public/ (calculemus.wasm + wasm_exec.js).
    command: "npx vite --port 5199 --strictPort",
    port: 5199,
    reuseExistingServer: true,
  },
});
