// The universe store: one reactive document, autosaved to localStorage on
// every change, re-evaluated through the WASM engine on a debounce.
// Guardrail R2: nothing exists until it persists — persistence is here, in
// the first milestone, not a future feature.
import { reactive, ref, watch } from "vue";
import type { Argument, Formula, Op, Statement, Universe, Verdicts } from "./types";
import { evaluate } from "./engine";

// v2 key: the frozen Boolean edition keeps "calculemus.universe" untouched.
const STORAGE_KEY = "calculemus.v2.universe";

export function emptyUniverse(): Universe {
  return {
    version: 1,
    title: "Untitled universe",
    statements: [],
    formulas: [],
    assertions: [],
    arguments: [],
    scenarios: [],
    layout: {},
  };
}

function loadSaved(): Universe | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const u = JSON.parse(raw) as Universe;
    if (typeof u.version !== "number" || !Array.isArray(u.statements)) return null;
    return u;
  } catch {
    return null;
  }
}

export const universe = reactive<Universe>(loadSaved() ?? emptyUniverse());
export const verdicts = ref<Verdicts | null>(null);
export const engineError = ref("");
export const activeScenario = ref("");

// Inspector selection: a statement, formula, or argument id. Clicking the
// selected row again deselects.
export const selected = ref<string | null>(null);

// Read-only mode: viewing someone else's shared universe. Verdicts and
// scenario switching work; edits and persistence are off, so the viewer's
// own saved universe is never clobbered.
export const readOnly = ref(false);

export function select(id: string): void {
  selected.value = selected.value === id ? null : id;
}

let timer: ReturnType<typeof setTimeout> | undefined;
async function evaluateNow() {
  try {
    verdicts.value = await evaluate(universe, activeScenario.value);
    engineError.value = "";
  } catch (err) {
    verdicts.value = null;
    engineError.value = err instanceof Error ? err.message : String(err);
  }
}

watch(
  [universe, activeScenario],
  () => {
    // Persist synchronously: a debounced save loses the last change when the
    // tab closes or reloads inside the window (guardrail R2). The document is
    // small; only the engine evaluation is worth debouncing.
    if (!readOnly.value) localStorage.setItem(STORAGE_KEY, JSON.stringify(universe));
    clearTimeout(timer);
    timer = setTimeout(evaluateNow, 250);
  },
  { deep: true, immediate: true },
);

// ---- id allocation ---------------------------------------------------------

function nextId(prefix: string): string {
  const taken = new Set([
    ...universe.statements.map((s) => s.id),
    ...(universe.formulas ?? []).map((f) => f.id),
    ...(universe.arguments ?? []).map((a) => a.id),
  ]);
  for (let n = 1; ; n++) {
    const id = `${prefix}${n}`;
    if (!taken.has(id)) return id;
  }
}

// ---- mutations -------------------------------------------------------------

export function addStatement(s: Omit<Statement, "id">): void {
  universe.statements.push({ id: nextId("s"), ...s });
}

export function addFormula(op: Op, args: string[]): void {
  (universe.formulas ??= []).push({ id: nextId("f"), op, args });
}

export function addArgument(title: string, premises: string[], conclusion: string): void {
  (universe.arguments ??= []).push({ id: nextId("a"), title, premises, conclusion });
}

function currentScenario() {
  return (universe.scenarios ?? []).find((s) => s.name === activeScenario.value) ?? null;
}

// Assertion state is scenario-aware: while a scenario is active, its toggles
// shadow the base assertions, and edits write to the scenario — the base
// universe stays untouched. Mirrors core's activeAssertions resolution.
export function isAsserted(ref: string): boolean {
  const sc = currentScenario();
  if (sc && ref in sc.toggles) return sc.toggles[ref];
  return (universe.assertions ?? []).some((a) => a.formula === ref && a.active);
}

export function setAsserted(ref: string, on: boolean): void {
  if (readOnly.value) return;
  const sc = currentScenario();
  if (sc) {
    sc.toggles[ref] = on;
    return;
  }
  const list = (universe.assertions ??= []);
  const existing = list.find((a) => a.formula === ref);
  if (on) {
    if (existing) existing.active = true;
    else list.push({ formula: ref, active: true, source: "hand" });
  } else if (existing) {
    list.splice(list.indexOf(existing), 1);
  }
}

export function addScenario(name: string): string {
  if (readOnly.value) return "Read-only view — copy to your workspace first.";
  const trimmed = name.trim();
  if (!trimmed) return "A scenario needs a name.";
  if ((universe.scenarios ?? []).some((s) => s.name === trimmed)) {
    return `A scenario named “${trimmed}” already exists.`;
  }
  (universe.scenarios ??= []).push({ name: trimmed, toggles: {} });
  activeScenario.value = trimmed;
  return "";
}

export function removeScenario(name: string): void {
  universe.scenarios = (universe.scenarios ?? []).filter((s) => s.name !== name);
  if (activeScenario.value === name) activeScenario.value = "";
}

// referencedBy lists what still points at an id; deletion is only allowed
// when nothing does, so the document can never hold a dangling ref.
export function referencedBy(id: string): string[] {
  const users: string[] = [];
  for (const f of universe.formulas ?? []) {
    if (f.args.includes(id)) users.push(f.id);
  }
  for (const a of universe.arguments ?? []) {
    if (a.premises.includes(id) || a.conclusion === id) users.push(a.id);
  }
  for (const sc of universe.scenarios ?? []) {
    if (id in sc.toggles) users.push(`scenario ${sc.name}`);
  }
  return users;
}

export function removeRef(id: string): void {
  if (readOnly.value || referencedBy(id).length > 0) return;
  if (selected.value === id) selected.value = null;
  universe.statements = spliceById(universe.statements, id);
  universe.formulas = spliceById(universe.formulas ?? [], id);
  universe.arguments = spliceById(universe.arguments ?? [], id);
  universe.assertions = (universe.assertions ?? []).filter((a) => a.formula !== id);
  if (universe.layout) delete universe.layout[id];
}

function spliceById<T extends { id: string }>(list: T[], id: string): T[] {
  return list.filter((item) => item.id !== id);
}

export function resetUniverse(): void {
  Object.assign(universe, emptyUniverse());
  universe.statements = [];
  universe.formulas = [];
  universe.assertions = [];
  universe.arguments = [];
  universe.scenarios = [];
  activeScenario.value = "";
}

// ---- canvas layout ----------------------------------------------------------

export function setLayout(id: string, x: number, y: number): void {
  (universe.layout ??= {})[id] = { x: Math.round(x), y: Math.round(y) };
}

// Forget all hand-placed positions; the canvas falls back to auto-placement.
export function clearLayout(): void {
  universe.layout = {};
}

// ---- import / export -------------------------------------------------------

export function exportUniverse(): void {
  const blob = new Blob([JSON.stringify(universe, null, 2)], { type: "application/json" });
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = `${universe.title.replace(/[^\w-]+/g, "-").toLowerCase() || "universe"}.json`;
  a.click();
  URL.revokeObjectURL(a.href);
}

// ---- sharing (M5) -----------------------------------------------------------

// shareUniverse publishes an immutable snapshot and returns its share URL.
export async function shareUniverse(): Promise<string> {
  const res = await fetch("/api/universes", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(universe),
  });
  const body = (await res.json()) as { path?: string; error?: string };
  if (!res.ok || !body.path) {
    throw new Error(body.error ?? `share failed (${res.status})`);
  }
  return `${location.origin}${body.path}`;
}

// loadShared fetches a shared universe into read-only view. Returns "" on
// success, an error message otherwise.
export async function loadShared(id: string): Promise<string> {
  let res: Response;
  try {
    res = await fetch(`/api/universes/${id}`);
  } catch {
    return "Could not reach the sharing server.";
  }
  if (!res.ok) return "That shared universe doesn't exist (or the link is stale).";
  const parsed = (await res.json()) as Universe;
  readOnly.value = true;
  activeScenario.value = "";
  selected.value = null;
  Object.assign(universe, emptyUniverse(), parsed);
  return "";
}

// copyToWorkspace adopts the shared universe as the viewer's own: editing
// and persistence come back on, and the URL returns to the workbench.
export function copyToWorkspace(): void {
  readOnly.value = false;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(universe));
  history.replaceState(null, "", "/");
}

export function importUniverse(text: string): string {
  let parsed: Universe;
  try {
    parsed = JSON.parse(text) as Universe;
  } catch {
    return "That file is not valid JSON.";
  }
  if (typeof parsed.version !== "number" || !Array.isArray(parsed.statements)) {
    return "That file is not a Calculemus universe (missing version or statements).";
  }
  resetUniverse();
  Object.assign(universe, parsed);
  return "";
}
