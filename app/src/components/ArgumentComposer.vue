<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { universe, addArgument, updateArgument, editing } from "../store";
import { renderRef } from "../render";

const title = ref("");
const premises = ref<string[]>([]);
const conclusion = ref("");

const refs = computed(() => [
  ...universe.statements.map((s) => s.id),
  ...(universe.formulas ?? []).map((f) => f.id),
]);

const editingArgument = computed(
  () => (universe.arguments ?? []).find((a) => a.id === editing.value) ?? null,
);

watch(editingArgument, (a) => {
  if (!a) return;
  title.value = a.title;
  premises.value = [...a.premises];
  conclusion.value = a.conclusion;
});

function reset() {
  title.value = "";
  premises.value = [];
  conclusion.value = "";
}

function cancelEdit() {
  editing.value = null;
  reset();
}

const complete = computed(
  () => title.value.trim() !== "" && premises.value.length > 0 && conclusion.value !== "",
);

function togglePremise(id: string, on: boolean) {
  premises.value = on
    ? [...premises.value, id]
    : premises.value.filter((p) => p !== id);
}

function submit() {
  if (!complete.value) return;
  if (editingArgument.value) {
    updateArgument(editingArgument.value.id, title.value.trim(), [...premises.value], conclusion.value);
    editing.value = null;
  } else {
    addArgument(title.value.trim(), [...premises.value], conclusion.value);
  }
  reset();
}
</script>

<template>
  <form v-if="refs.length >= 2" class="composer" @submit.prevent="submit">
    <span v-if="editingArgument" class="small editing-note">Editing “{{ editingArgument.title }}”.</span>
    <input v-model="title" placeholder="argument title" aria-label="Argument title" />
    <fieldset>
      <legend class="muted small">premises</legend>
      <label v-for="r in refs" :key="r" class="small premise">
        <input
          type="checkbox"
          :checked="premises.includes(r)"
          @change="togglePremise(r, ($event.target as HTMLInputElement).checked)"
        />
        {{ renderRef(universe, r) }}
      </label>
    </fieldset>
    <label class="small conclusion">
      ∴
      <select v-model="conclusion" aria-label="Conclusion">
        <option value="" disabled>conclusion…</option>
        <option v-for="r in refs" :key="r" :value="r">{{ renderRef(universe, r) }}</option>
      </select>
    </label>
    <span class="buttons">
      <button class="primary" type="submit" :disabled="!complete">
        {{ editingArgument ? "Save argument" : "Add argument" }}
      </button>
      <button v-if="editingArgument" type="button" @click="cancelEdit">Cancel</button>
    </span>
  </form>
  <p v-else class="muted small">An argument needs at least two things to relate.</p>
</template>

<style scoped>
.composer {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  margin-top: 0.6rem;
  padding-top: 0.6rem;
  border-top: 1px dashed var(--rule);
}
fieldset {
  border: 1px solid var(--rule);
  border-radius: 5px;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}
.premise,
.conclusion {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.conclusion select {
  flex: 1;
}
.buttons {
  display: flex;
  gap: 0.4rem;
}
.editing-note {
  color: var(--accent);
}
</style>
