<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { Op } from "../types";
import { ALL_OPS, BINARY_OPS, UNARY_OPS } from "../types";
import { universe, addFormula } from "../store";
import { renderRef } from "../render";

const op = ref<Op>("IMPLIES");
const args = ref<string[]>(["", ""]);

const refs = computed(() => [
  ...universe.statements.map((s) => s.id),
  ...(universe.formulas ?? []).map((f) => f.id),
]);

const arity = computed(() =>
  UNARY_OPS.includes(op.value) ? 1 : BINARY_OPS.includes(op.value) ? 2 : -1,
);

watch(arity, (n) => {
  if (n === 1) args.value = args.value.slice(0, 1);
  else if (args.value.length < 2) args.value = [...args.value, ""];
  if (args.value.length === 0) args.value = [""];
});

const complete = computed(
  () => args.value.length >= (arity.value === 1 ? 1 : 2) && args.value.every((a) => a !== ""),
);

const preview = computed(() => {
  if (!complete.value) return "";
  if (op.value === "NOT") return `NOT (${renderRef(universe, args.value[0])})`;
  return `(${args.value.map((a) => renderRef(universe, a)).join(` ${op.value} `)})`;
});

function submit() {
  if (!complete.value) return;
  addFormula(op.value, [...args.value]);
  args.value = arity.value === 1 ? [""] : ["", ""];
}
</script>

<template>
  <form v-if="refs.length > 0" class="composer" @submit.prevent="submit">
    <select v-model="op" aria-label="Connective">
      <option v-for="o in ALL_OPS" :key="o" :value="o">{{ o }}</option>
    </select>
    <select
      v-for="(_, i) in args"
      :key="i"
      v-model="args[i]"
      :aria-label="`Term ${i + 1}`"
    >
      <option value="" disabled>term {{ i + 1 }}…</option>
      <option v-for="r in refs" :key="r" :value="r">{{ renderRef(universe, r) }}</option>
    </select>
    <button
      v-if="arity === -1"
      type="button"
      title="Add another term"
      @click="args.push('')"
    >
      + term
    </button>
    <button class="primary" type="submit" :disabled="!complete">Add formula</button>
    <span v-if="preview" class="muted small preview">{{ preview }}</span>
  </form>
</template>

<style scoped>
.composer {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  align-items: center;
  margin-top: 0.6rem;
  padding-top: 0.6rem;
  border-top: 1px dashed var(--rule);
}
.composer select {
  max-width: 16rem;
}
.preview {
  flex-basis: 100%;
}
</style>
