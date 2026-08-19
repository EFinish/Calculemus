<script setup lang="ts">
import { computed } from "vue";
import { universe, verdicts, isAsserted, setAsserted, referencedBy, removeRef, select, selected } from "../store";
import { renderRef } from "../render";
import StatementComposer from "./StatementComposer.vue";
import FormulaComposer from "./FormulaComposer.vue";
import ArgumentComposer from "./ArgumentComposer.vue";

// Truth-state badge for a ref: what the user asserted, or what the engine
// forced. Entailment marks are the engine talking — styled "derived".
const truthOf = computed(() => (ref: string): "asserted" | "true" | "false" | null => {
  if (isAsserted(ref)) return "asserted";
  if (verdicts.value?.entailedTrue?.includes(ref)) return "true";
  if (verdicts.value?.entailedFalse?.includes(ref)) return "false";
  return null;
});

const inCore = (ref: string) => verdicts.value?.unsatCore?.includes(ref) ?? false;

function deleteTitle(id: string): string {
  const users = referencedBy(id);
  return users.length ? `In use by: ${users.join(", ")}` : "Delete";
}
</script>

<template>
  <div class="stack">
    <section class="card">
      <h2>Statements</h2>
      <p v-if="universe.statements.length === 0" class="muted small">
        The atoms of your universe. Compose one below.
      </p>
      <div v-for="s in universe.statements" :key="s.id" class="row selectable" :class="{ selected: selected === s.id }" @click="select(s.id)">
        <label class="assert small" :title="'Assert: commit to this being true'" @click.stop>
          <input
            type="checkbox"
            :checked="isAsserted(s.id)"
            @change="setAsserted(s.id, ($event.target as HTMLInputElement).checked)"
          />
          assert
        </label>
        <span class="grow" :class="{ conflict: inCore(s.id) }">{{ s.text }}</span>
        <span v-if="inCore(s.id)" class="badge bad">in conflict</span>
        <span v-else-if="truthOf(s.id) === 'true'" class="badge ok">⊨ true</span>
        <span v-else-if="truthOf(s.id) === 'false'" class="badge bad">⊨ false</span>
        <button
          class="small"
          :disabled="referencedBy(s.id).length > 0"
          :title="deleteTitle(s.id)"
          @click.stop="removeRef(s.id)"
        >
          ✕
        </button>
      </div>
      <StatementComposer />
    </section>

    <section class="card">
      <h2>Formulas</h2>
      <p v-if="(universe.formulas ?? []).length === 0" class="muted small">
        Connect statements with NOT, AND, OR, IF-THEN… Needs at least one statement.
      </p>
      <div v-for="f in universe.formulas" :key="f.id" class="row selectable" :class="{ selected: selected === f.id }" @click="select(f.id)">
        <label class="assert small" @click.stop>
          <input
            type="checkbox"
            :checked="isAsserted(f.id)"
            @change="setAsserted(f.id, ($event.target as HTMLInputElement).checked)"
          />
          assert
        </label>
        <span class="grow" :class="{ conflict: inCore(f.id) }">{{ renderRef(universe, f.id) }}</span>
        <span v-if="inCore(f.id)" class="badge bad">in conflict</span>
        <span
          v-else-if="verdicts?.vacuous?.includes(f.id)"
          class="badge info"
          title="True only because its IF-part is forced false — it says nothing here"
        >
          vacuous
        </span>
        <button
          class="small"
          :disabled="referencedBy(f.id).length > 0"
          :title="deleteTitle(f.id)"
          @click.stop="removeRef(f.id)"
        >
          ✕
        </button>
      </div>
      <FormulaComposer />
    </section>

    <section class="card">
      <h2>Arguments</h2>
      <p v-if="(universe.arguments ?? []).length === 0" class="muted small">
        Premises ⊢ conclusion. Validity is computed, never declared.
      </p>
      <div v-for="a in universe.arguments" :key="a.id" class="row selectable" :class="{ selected: selected === a.id }" @click="select(a.id)">
        <span class="grow">
          <strong>{{ a.title }}</strong>
          <span class="muted small">
            — {{ a.premises.map((p) => renderRef(universe, p)).join("; ") }} ∴
            {{ renderRef(universe, a.conclusion) }}
          </span>
        </span>
        <span
          v-if="verdicts?.arguments?.find((v) => v.id === a.id)?.form"
          class="badge info"
          title="A recognized argument form — decorative; the verdict comes from semantics, not the name"
        >
          {{ verdicts?.arguments?.find((v) => v.id === a.id)?.form }}
        </span>
        <span
          v-if="verdicts?.arguments?.find((v) => v.id === a.id)?.valid"
          class="badge ok"
        >
          valid
        </span>
        <span v-else-if="verdicts" class="badge bad">invalid</span>
        <button class="small" title="Delete" @click.stop="removeRef(a.id)">✕</button>
      </div>
      <ArgumentComposer />
    </section>
  </div>
</template>

<style scoped>
.stack {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.assert {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--muted);
  white-space: nowrap;
}
.conflict {
  color: var(--false);
}
.selectable {
  cursor: pointer;
}
.selectable.selected {
  background: var(--accent-soft);
  box-shadow: inset 2px 0 0 var(--accent);
}
</style>
