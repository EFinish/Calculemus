<script setup lang="ts">
// Guided composition (apriorio's best invention): pick the pieces, never
// type raw logic. M6 grows the grammar one slot: subject phrase + verb +
// object phrase. Defaults ("all of … is …") are exactly the pre-M6 copular
// form, so nothing changes until you reach for "the" or a verb.
import { computed, ref } from "vue";
import type { Qualifier, Quantifier, Statement } from "../types";
import { addStatement } from "../store";

// "THE" marks an individual phrase; the rest are kind quantifiers.
type PhraseMode = Quantifier | "THE";
type VerbMode = "IS" | "IS_NOT" | "DOES" | "DOES_NOT";

const subjectMode = ref<PhraseMode>("ALL");
const subject = ref("");
const verbMode = ref<VerbMode>("IS");
const verb = ref("");
const objectMode = ref<PhraseMode>("THE");
const object = ref("");

const isVerb = computed(() => verbMode.value === "DOES" || verbMode.value === "DOES_NOT");
const negated = computed(() => verbMode.value === "IS_NOT" || verbMode.value === "DOES_NOT");

const PHRASE_WORDS: Record<PhraseMode, string> = {
  ALL: "all of",
  SOME: "some of",
  NONE: "none of",
  THE: "the",
};

function phraseText(mode: PhraseMode, name: string): string {
  return `${PHRASE_WORDS[mode]} ${name.trim()}`;
}

const preview = computed(() => {
  if (!subject.value.trim() || !object.value.trim()) return "";
  const subj = phraseText(subjectMode.value, subject.value);
  if (!isVerb.value) {
    return `${subj} ${negated.value ? "is not" : "is"} ${object.value.trim()}`;
  }
  if (!verb.value.trim()) return "";
  const verbPart = negated.value ? `does not ${verb.value.trim()}` : `${verb.value.trim()}s`;
  return `${subj} ${verbPart} ${phraseText(objectMode.value, object.value)}`;
});

function submit() {
  if (!preview.value) return;
  const s: Omit<Statement, "id"> = {
    text: preview.value,
    subject: subject.value.trim(),
    predicate: object.value.trim(),
    qualifier: negated.value ? "IS_NOT" : "IS",
  };
  if (subjectMode.value === "THE") s.subjectIsIndividual = true;
  else s.quantifier = subjectMode.value;
  if (isVerb.value) {
    s.verb = verb.value.trim();
    if (objectMode.value === "THE") s.objectIsIndividual = true;
    else s.objectQuantifier = objectMode.value;
  }
  addStatement(s);
  subject.value = "";
  object.value = "";
  verb.value = "";
}
</script>

<template>
  <form class="composer" @submit.prevent="submit">
    <select v-model="subjectMode" aria-label="Quantifier">
      <option value="ALL">all of</option>
      <option value="SOME">some of</option>
      <option value="NONE">none of</option>
      <option value="THE">the</option>
    </select>
    <input v-model="subject" placeholder="subject — the ball" aria-label="Subject" />
    <select v-model="verbMode" aria-label="Qualifier">
      <option value="IS">is</option>
      <option value="IS_NOT">is not</option>
      <option value="DOES">does…</option>
      <option value="DOES_NOT">does not…</option>
    </select>
    <input
      v-if="isVerb"
      v-model="verb"
      placeholder="verb — throw"
      aria-label="Verb"
    />
    <select v-if="isVerb" v-model="objectMode" aria-label="Object quantifier">
      <option value="THE">the</option>
      <option value="ALL">all of</option>
      <option value="SOME">some of</option>
      <option value="NONE">none of</option>
    </select>
    <input
      v-model="object"
      :placeholder="isVerb ? 'object — the ball' : 'predicate — red'"
      aria-label="Predicate"
    />
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
  min-width: 7rem;
}
.preview {
  flex-basis: 100%;
}
</style>
