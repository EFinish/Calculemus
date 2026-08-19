<script setup lang="ts">
import { ref } from "vue";
import {
  universe,
  verdicts,
  engineError,
  activeScenario,
  resetUniverse,
  exportUniverse,
  importUniverse,
} from "./store";
import LibraryPane from "./components/LibraryPane.vue";
import CanvasPane from "./components/CanvasPane.vue";
import VerdictPane from "./components/VerdictPane.vue";
import SelectionInspector from "./components/SelectionInspector.vue";

const fileInput = ref<HTMLInputElement>();
const importError = ref("");

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
</script>

<template>
  <header>
    <span class="wordmark">∴ Calculemus</span>
    <input v-model="universe.title" class="title" aria-label="Universe title" />
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
    <span class="spacer"></span>
    <button @click="onNew">New</button>
    <button @click="exportUniverse">Export</button>
    <button @click="fileInput?.click()">Import</button>
    <input ref="fileInput" type="file" accept=".json,application/json" hidden @change="onImportFile" />
  </header>
  <p v-if="importError" class="import-error">{{ importError }}</p>

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
