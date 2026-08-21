<script setup lang="ts">
// Guided composition (apriorio's best invention): pick the pieces, never
// type raw logic. M6 grows the grammar one slot: subject phrase + verb +
// object phrase. Defaults ("all of … is …") are exactly the pre-M6 copular
// form, so nothing changes until you reach for "the" or a verb.
import { computed, ref, watch } from "vue";
import type { Quantifier, Statement } from "../types";
import { universe, addStatement, updateStatement, editing } from "../store";

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

// "is" typed as a verb would create an uninterpreted relation named "is" —
// no membership or identity meaning — which is almost never what a human
// wants. Refuse it and point at the copula (the philosophers' three-ways-
// ambiguous "is" strikes again).
const COPULA_WORDS = ["is", "are", "be", "am", "was", "were", "being", "been"];
const copulaAsVerb = computed(
  () => isVerb.value && COPULA_WORDS.includes(verb.value.trim().toLowerCase()),
);

const editingStatement = computed(
  () => universe.statements.find((s) => s.id === editing.value) ?? null,
);

watch(editingStatement, (s) => {
  if (!s) return;
  subjectMode.value = s.subjectIsIndividual ? "THE" : (s.quantifier ?? "ALL");
  subject.value = s.subject ?? "";
  const neg = s.qualifier === "IS_NOT";
  verbMode.value = s.verb ? (neg ? "DOES_NOT" : "DOES") : neg ? "IS_NOT" : "IS";
  verb.value = s.verb ?? "";
  objectMode.value = s.objectIsIndividual ? "THE" : (s.objectQuantifier ?? "THE");
  object.value = s.predicate ?? "";
});

function reset() {
  subject.value = "";
  object.value = "";
  verb.value = "";
}

function cancelEdit() {
  editing.value = null;
  reset();
}
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
  if (copulaAsVerb.value) return "";
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
  if (editingStatement.value) {
    updateStatement(editingStatement.value.id, s);
    editing.value = null;
  } else {
    addStatement(s);
  }
  reset();
}
</script>

<template>
  <form class="composer" @submit.prevent="submit">
    <span v-if="editingStatement" class="small editing-note">
      Editing “{{ editingStatement.text }}” — every reference updates with it.
    </span>
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
    <span v-if="copulaAsVerb" class="small copula-warning">
      “{{ verb.trim() }}” as a verb would be an uninterpreted relation with no
      membership meaning — for “{{ subject.trim() || "the ball" }} is {{ object.trim() || "red" }}”,
      pick <strong>is</strong> in the dropdown instead (no verb needed).
    </span>
    <button class="primary" type="submit" :disabled="!preview">
      {{ editingStatement ? "Save statement" : "Add statement" }}
    </button>
    <button v-if="editingStatement" type="button" @click="cancelEdit">Cancel</button>
    <span v-if="preview" class="muted small preview">“{{ preview }}”</span>
    <span class="muted small hint">
      Tip: pick <strong>the</strong> and <strong>does…</strong> for relational statements —
      then the engine can connect the object of one statement to the subject of another
      (“the boy throws <em>the ball</em>” + “<em>the ball</em> is red” derives
      “the boy throws some of red”).
    </span>
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
.hint {
  flex-basis: 100%;
  opacity: 0.85;
}
.editing-note {
  flex-basis: 100%;
  color: var(--accent);
}
.copula-warning {
  flex-basis: 100%;
  color: var(--false);
}
</style>
