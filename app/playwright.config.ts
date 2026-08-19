import { defineConfig, devices } from "@playwright/test";

// Firefox on purpose: the e2e dogfood runs without any browser-extension
// pairing, so it works headless and in CI.
export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  // Desktop viewport so the three-pane layout applies and the canvas is on
  // screen — below 1280px it reflows under the library, where raw mouse
  // coordinates for canvas drags would land outside the viewport.
  use: { baseURL: "http://localhost:5199" },
  // Viewport must live in the project's `use` — the device spread carries its
  // own 1280×720 viewport and project-level use overrides the global one.
  projects: [
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"], viewport: { width: 1600, height: 1000 } },
    },
  ],
  webServer: {
    // Assumes `make wasm` has populated public/ (calculemus.wasm + wasm_exec.js).
    command: "npx vite --port 5199 --strictPort",
    port: 5199,
    reuseExistingServer: true,
  },
});
