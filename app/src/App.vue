<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  universe,
  verdicts,
  engineError,
  activeScenario,
  resetUniverse,
  exportUniverse,
  importUniverse,
  addScenario,
  removeScenario,
  readOnly,
  shareUniverse,
  loadShared,
  copyToWorkspace,
} from "./store";
import LibraryPane from "./components/LibraryPane.vue";
import CanvasPane from "./components/CanvasPane.vue";
import VerdictPane from "./components/VerdictPane.vue";
import SelectionInspector from "./components/SelectionInspector.vue";

const fileInput = ref<HTMLInputElement>();
const importError = ref("");
const shareLink = ref("");
const shareError = ref("");

// A /u/{id} URL is a shared universe: load it read-only.
onMounted(async () => {
  const m = location.pathname.match(/^\/u\/([a-z0-9]{10})$/);
  if (m) importError.value = await loadShared(m[1]);
});

async function onShare() {
  shareError.value = "";
  try {
    shareLink.value = await shareUniverse();
    await navigator.clipboard?.writeText(shareLink.value).catch(() => {});
  } catch (err) {
    shareError.value = err instanceof Error ? err.message : String(err);
  }
}

async function onImportFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0];
  if (!file) return;
  importError.value = importUniverse(await file.text());
  (e.target as HTMLInputElement).value = "";
}

function onNew() {
  if (window.confirm("Start a new universe? The current one is only kept if you exported it.")) {
    resetUniverse();
  }
}

function onNewScenario() {
  const name = window.prompt("Scenario name — a counterfactual world to explore:");
  if (name === null) return;
  const err = addScenario(name);
  if (err) window.alert(err);
}

function onDeleteScenario() {
  if (window.confirm(`Delete scenario “${activeScenario.value}”? The base universe is unaffected.`)) {
    removeScenario(activeScenario.value);
  }
}
</script>

<template>
  <header>
    <span class="wordmark">∴ Calculemus</span>
    <input v-model="universe.title" class="title" aria-label="Universe title" :readonly="readOnly" />
    <span v-if="engineError" class="badge bad" :title="engineError">engine error</span>
    <span v-else-if="!verdicts" class="badge muted">evaluating…</span>
    <span v-else-if="verdicts.consistent" class="badge ok">consistent</span>
    <span v-else class="badge bad">contradictory</span>
    <select
      v-if="(universe.scenarios ?? []).length > 0"
      v-model="activeScenario"
      aria-label="Scenario"
    >
      <option value="">live assertions</option>
      <option v-for="sc in universe.scenarios" :key="sc.name" :value="sc.name">
        scenario: {{ sc.name }}
      </option>
    </select>
    <button v-if="!readOnly" title="Create a counterfactual world: toggles you flip while it's active edit the scenario, not the base universe" @click="onNewScenario">+ scenario</button>
    <button v-if="!readOnly && activeScenario" title="Delete this scenario" @click="onDeleteScenario">✕ scenario</button>
    <span v-if="readOnly" class="badge info">shared · read-only</span>
    <span class="spacer"></span>
    <button v-if="readOnly" class="primary" @click="copyToWorkspace">Copy to my workspace</button>
    <template v-else>
      <button @click="onNew">New</button>
      <button title="Publish an immutable snapshot and get a link anyone can open" @click="onShare">Share</button>
      <button @click="exportUniverse">Export</button>
      <button @click="fileInput?.click()">Import</button>
    </template>
    <input ref="fileInput" type="file" accept=".json,application/json" hidden @change="onImportFile" />
  </header>
  <p v-if="importError" class="import-error">{{ importError }}</p>
  <p v-if="shareError" class="import-error">{{ shareError }}</p>
  <p v-if="shareLink" class="share-link">
    Shared — anyone with this link can explore (copied to clipboard):
    <input class="link" readonly :value="shareLink" aria-label="Share link" @focus="($event.target as HTMLInputElement).select()" />
  </p>

  <main>
    <LibraryPane />
    <CanvasPane />
    <div class="right">
      <VerdictPane />
      <SelectionInspector />
    </div>
  </main>
</template>

<style scoped>
header {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 1rem;
  border-bottom: 1px solid var(--rule);
  background: var(--surface);
  position: sticky;
  top: 0;
  z-index: 2;
}
.wordmark {
  font-weight: 700;
  color: var(--accent);
  white-space: nowrap;
}
.title {
  font-weight: 600;
  min-width: 0;
  width: 18rem;
}
.spacer {
  flex: 1;
}
.import-error {
  margin: 0;
  padding: 0.5rem 1rem;
  background: var(--false-soft);
  color: var(--false);
}
.share-link {
  margin: 0;
  padding: 0.5rem 1rem;
  background: var(--accent-soft);
  color: var(--ink);
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 0.88rem;
}
.share-link .link {
  flex: 1;
  font-family: ui-monospace, monospace;
  font-size: 0.8rem;
}
/* DESIGN §6: library | canvas | inspector — a workbench, not a site. */
main {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1.6fr) minmax(0, 1fr);
  gap: 1rem;
  padding: 1rem;
  align-items: start;
}
@media (max-width: 1280px) {
  main {
    grid-template-columns: minmax(0, 3fr) minmax(0, 2fr);
  }
  main > :nth-child(2) {
    grid-column: 1 / -1;
    order: 3;
  }
}
.right {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  position: sticky;
  top: 4rem;
}
@media (max-width: 900px) {
  main {
    grid-template-columns: 1fr;
  }
  .right {
    position: static;
  }
}
</style>
