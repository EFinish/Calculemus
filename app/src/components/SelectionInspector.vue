<script setup lang="ts">
// The M2 inspector: context for whatever is selected in the library.
// Statements get their truth state and where they're used; formulas get
// their assertion state and vacuousness explained; arguments get their
// verdict — and, when invalid, the concrete countermodel: a world where
// every premise holds and the conclusion fails.
import { computed } from "vue";
import { universe, verdicts, selected, isAsserted, referencedBy } from "../store";
import { renderRef } from "../render";

const statement = computed(() =>
  universe.statements.find((s) => s.id === selected.value),
);
const formula = computed(() =>
  (universe.formulas ?? []).find((f) => f.id === selected.value),
);
const argument = computed(() =>
  (universe.arguments ?? []).find((a) => a.id === selected.value),
);

type TruthState = "asserted" | "entailed-true" | "entailed-false" | "undetermined";
const truthState = computed((): TruthState => {
  const id = selected.value!;
  if (isAsserted(id)) return "asserted";
  if (verdicts.value?.entailedTrue?.includes(id)) return "entailed-true";
  if (verdicts.value?.entailedFalse?.includes(id)) return "entailed-false";
  return "undetermined";
});

const usedBy = computed(() => referencedBy(selected.value!));

const argVerdict = computed(() =>
  verdicts.value?.arguments?.find((v) => v.id === selected.value),
);

// Chains touching the selected argument: what feeds it, what it feeds.
const chainsIn = computed(() =>
  (verdicts.value?.edges ?? []).filter((e) => e.type === "chains" && e.to === selected.value),
);
const chainsOut = computed(() =>
  (verdicts.value?.edges ?? []).filter((e) => e.type === "chains" && e.from === selected.value),
);
const argTitle = (id: string) =>
  (universe.arguments ?? []).find((a) => a.id === id)?.title ?? id;
</script>

<template>
  <section v-if="selected && (statement || formula || argument)" class="card">
    <h2>Inspector</h2>

    <template v-if="statement">
      <p><strong>{{ statement.text }}</strong></p>
      <p v-if="statement.subject" class="muted small">
        {{ statement.subjectIsIndividual ? "THE" : statement.quantifier }} ·
        {{ statement.subject }} ·
        {{ statement.verb ? (statement.qualifier === "IS" ? "DOES" : "DOES NOT") + " " + statement.verb : statement.qualifier }} ·
        {{ statement.objectIsIndividual ? "THE" : (statement.objectQuantifier ?? "") }}
        {{ statement.predicate }}
      </p>
      <p class="small">
        <span v-if="truthState === 'asserted'" class="badge info">asserted</span>
        <span v-else-if="truthState === 'entailed-true'" class="badge ok">⊨ true</span>
        <span v-else-if="truthState === 'entailed-false'" class="badge bad">⊨ false</span>
        <span v-else class="badge muted">undetermined</span>
        <span class="muted">
          {{
            truthState === "asserted"
              ? " — you committed to this being true."
              : truthState === "entailed-true"
                ? " — your assertions force this true; it cannot be false in any world where they hold."
                : truthState === "entailed-false"
                  ? " — your assertions force this false; it cannot be true in any world where they hold."
                  : " — your assertions don't force it either way. Epistemic, not a third truth value."
          }}
        </span>
      </p>
    </template>

    <template v-else-if="formula">
      <p><strong>{{ renderRef(universe, formula.id) }}</strong></p>
      <p class="small">
        <span v-if="isAsserted(formula.id)" class="badge info">asserted</span>
        <span v-else class="badge muted">not asserted</span>
        <span
          v-if="verdicts?.vacuous?.includes(formula.id)"
          class="badge info"
        >vacuous</span>
      </p>
      <p v-if="verdicts?.vacuous?.includes(formula.id)" class="muted small">
        This conditional holds only vacuously: your assertions force its IF-part
        false, so it never fires. True, but it says nothing here.
      </p>
    </template>

    <template v-else-if="argument">
      <p><strong>{{ argument.title }}</strong></p>
      <ol class="small premlist">
        <li v-for="p in argument.premises" :key="p">{{ renderRef(universe, p) }}</li>
      </ol>
      <p class="small">∴ {{ renderRef(universe, argument.conclusion) }}</p>

      <template v-if="argVerdict?.valid">
        <p class="small">
          <span class="badge ok">valid</span>
          <span class="muted">
            No countermodel exists{{ verdicts?.boundedDomain ? ` among worlds with at most ${verdicts.boundedDomain} things` : "" }}:
            in every {{ verdicts?.boundedDomain ? "such" : "possible" }} world where the premises
            hold, the conclusion holds too.
          </span>
        </p>
        <p v-if="argVerdict.form" class="small">
          <span class="badge info">{{ argVerdict.form }}</span>
          <span class="muted">
            — a recognized form. The name decorates the verdict; semantics
            decided it.
          </span>
        </p>
      </template>
      <template v-else-if="argVerdict">
        <p class="small">
          <span class="badge bad">invalid</span>
          <span class="muted">Here is a world where every premise holds and the conclusion fails:</span>
        </p>
        <table class="countermodel small">
          <tbody>
            <tr v-for="s in universe.statements" :key="s.id">
              <td class="tv" :class="argVerdict.countermodel?.[s.id] ? 'ok' : 'bad'">
                {{ argVerdict.countermodel?.[s.id] ? "true" : "false" }}
              </td>
              <td>{{ s.text }}</td>
            </tr>
          </tbody>
        </table>
      </template>

      <template v-if="chainsIn.length + chainsOut.length > 0">
        <h3 class="small muted">Chains</h3>
        <p v-for="e in chainsIn" :key="'in' + e.from" class="small">
          <span class="chain">⊢→⊢</span> fed by <strong>{{ argTitle(e.from) }}</strong>
        </p>
        <p v-for="e in chainsOut" :key="'out' + e.to" class="small">
          <span class="chain">⊢→⊢</span> feeds <strong>{{ argTitle(e.to) }}</strong>
        </p>
      </template>
    </template>

    <template v-if="!argument && usedBy.length > 0">
      <h3 class="small muted">Used by</h3>
      <p v-for="id in usedBy" :key="id" class="small">
        {{ id.startsWith("scenario ") ? id : renderRef(universe, id) }}
      </p>
    </template>
  </section>
  <section v-else class="card">
    <h2>Inspector</h2>
    <p class="muted small">Select a statement, formula, or argument to inspect it.</p>
  </section>
</template>

<style scoped>
h3 {
  margin: 0.9rem 0 0.2rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.72rem;
}
p {
  margin: 0.35rem 0;
}
.dim {
  opacity: 0.7;
}
.premlist {
  margin: 0.25rem 0;
  padding-left: 1.4rem;
}
.countermodel {
  border-collapse: collapse;
  margin-top: 0.35rem;
}
.countermodel td {
  padding: 0.2rem 0.6rem 0.2rem 0;
  border-bottom: 1px solid var(--rule);
}
.tv {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.tv.ok {
  color: var(--true);
}
.tv.bad {
  color: var(--false);
}
.chain {
  color: var(--accent);
  font-weight: 600;
}
</style>
