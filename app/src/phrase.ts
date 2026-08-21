import type { Statement } from "./types";

const QUANT_WORDS: Record<string, string> = {
  ALL: "all of",
  SOME: "some of",
  NONE: "none of",
};

function phrase(name: string, quantifier: string | undefined, individual: boolean | undefined): string {
  return individual ? `the ${name}` : `${QUANT_WORDS[quantifier ?? "ALL"]} ${name}`;
}

// statementText renders a statement's sentence from its grammar — the one
// source of truth for phrasing, shared by the composer and the discoveries
// panel.
export function statementText(s: Omit<Statement, "id" | "text">): string {
  const subj = phrase(s.subject ?? "", s.quantifier, s.subjectIsIndividual);
  if (!s.verb) {
    return `${subj} ${s.qualifier === "IS_NOT" ? "is not" : "is"} ${s.predicate}`;
  }
  const v = s.verb.trim();
  const conj = v.endsWith("s") ? v : `${v}s`;
  const verbPart = s.qualifier === "IS_NOT" ? `does not ${v}` : conj;
  return `${subj} ${verbPart} ${phrase(s.predicate ?? "", s.objectQuantifier, s.objectIsIndividual)}`;
}
