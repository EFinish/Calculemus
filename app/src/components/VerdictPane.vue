<script setup lang="ts">
// Minimal verdict readout — proves the WASM bridge live and seeds the M2
// inspector. The full inspector (countermodels, per-selection context,
// diagnosis guidance) is milestone M2.
import { computed } from "vue";
import { universe, verdicts, engineError, activeScenario, setAsserted } from "../store";
import { renderRef } from "../render";

const argTitle = (id: string) =>
  (universe.arguments ?? []).find((a) => a.id === id)?.title ?? id;

const chains = computed(
  () => verdicts.value?.edges?.filter((e) => e.type === "chains") ?? [],
);
</script>

<template>
  <div class="stack">
    <section class="card">
      <h2>Verdicts</h2>
      <p v-if="engineError" class="small error">{{ engineError }}</p>
      <p v-else-if="!verdicts" class="muted small">Waking the engine…</p>
      <template v-else-if="verdicts.consistent">
        <p class="small"><span class="badge ok">consistent</span> Your assertions can all hold together.</p>
        <template v-if="(verdicts.entailedTrue ?? []).length + (verdicts.entailedFalse ?? []).length > 0">
          <h3 class="small muted">Derived truths — forced by your assertions</h3>
          <div v-for="id in verdicts.entailedTrue" :key="id" class="row small">
            <span class="badge ok">⊨ true</span>
            <span class="grow">{{ renderRef(universe, id) }}</span>
          </div>
          <div v-for="id in verdicts.entailedFalse" :key="id" class="row small">
            <span class="badge bad">⊨ false</span>
            <span class="grow">{{ renderRef(universe, id) }}</span>
          </div>
        </template>
        <p v-else class="muted small">Nothing is forced yet — assert some statements or formulas.</p>
      </template>
      <template v-else>
        <p class="small">
          <span class="badge bad">contradictory</span>
          These assertions cannot coexist — drop or unassert any one:
        </p>
        <div v-for="id in verdicts.unsatCore" :key="id" class="row small">
          <span class="badge bad">✗</span>
          <span class="grow">{{ renderRef(universe, id) }}</span>
          <button
            class="small"
            :title="
              activeScenario
                ? 'Unassert within this scenario (the base universe is untouched)'
                : 'Unassert: stop holding this true'
            "
            @click="setAsserted(id, false)"
          >
            unassert
          </button>
        </div>
        <p class="muted small">
          Derived truths are suspended: a contradiction entails everything.
        </p>
      </template>
    </section>

    <section v-if="verdicts && chains.length > 0" class="card">
      <h2>Chained arguments</h2>
      <p class="muted small">Conclusions feeding premises — nobody drew these.</p>
      <div v-for="(e, i) in chains" :key="i" class="row small">
        <span class="grow">{{ argTitle(e.from) }} <span class="chain">⊢→⊢</span> {{ argTitle(e.to) }}</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
/* Stickiness lives on App.vue's .right wrapper, which stacks this pane
   with the inspector. */
.stack {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
h3 {
  margin: 0.8rem 0 0.2rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.72rem;
}
.error {
  color: var(--false);
}
.chain {
  color: var(--accent);
  font-weight: 600;
}
</style>
