import type { Universe } from "./types";

// renderRef turns a ref into readable text: statements by their sentence,
// formulas recursively by their connective. Mirrors the CLI's renderer.
export function renderRef(u: Universe, ref: string): string {
  const s = u.statements.find((s) => s.id === ref);
  if (s) return s.text;
  const f = (u.formulas ?? []).find((f) => f.id === ref);
  if (!f) return ref;
  if (f.op === "NOT") return `NOT (${renderRef(u, f.args[0])})`;
  return `(${f.args.map((a) => renderRef(u, a)).join(` ${f.op} `)})`;
}
