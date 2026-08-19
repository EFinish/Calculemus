<script setup lang="ts">
// Guided composition (apriorio's best invention): pick the pieces, never
// type raw logic. The sentence is generated live from the parts.
import { computed, ref } from "vue";
import type { Qualifier, Quantifier } from "../types";
import { addStatement } from "../store";

const subject = ref("");
const quantifier = ref<Quantifier>("ALL");
const predicate = ref("");
const qualifier = ref<Qualifier>("IS");

const QUANT_WORDS: Record<Quantifier, string> = {
  ALL: "all of",
  SOME: "some of",
  NONE: "none of",
};

const preview = computed(() => {
  if (!subject.value.trim() || !predicate.value.trim()) return "";
  const is = qualifier.value === "IS" ? "is" : "is not";
  return `${QUANT_WORDS[quantifier.value]} ${subject.value.trim()} ${is} ${predicate.value.trim()}`;
});

function submit() {
  if (!preview.value) return;
  addStatement({
    text: preview.value,
    subject: subject.value.trim(),
    quantifier: quantifier.value,
    predicate: predicate.value.trim(),
    qualifier: qualifier.value,
  });
  subject.value = "";
  predicate.value = "";
}
</script>

<template>
  <form class="composer" @submit.prevent="submit">
    <select v-model="quantifier" aria-label="Quantifier">
      <option value="ALL">all of</option>
      <option value="SOME">some of</option>
      <option value="NONE">none of</option>
    </select>
    <input v-model="subject" placeholder="subject — the ball" aria-label="Subject" />
    <select v-model="qualifier" aria-label="Qualifier">
      <option value="IS">is</option>
      <option value="IS_NOT">is not</option>
    </select>
    <input v-model="predicate" placeholder="predicate — red" aria-label="Predicate" />
    <button class="primary" type="submit" :disabled="!preview">Add statement</button>
    <span v-if="preview" class="muted small preview">“{{ preview }}”</span>
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
.composer input {
  flex: 1;
  min-width: 8rem;
}
.preview {
  flex-basis: 100%;
}
</style>
