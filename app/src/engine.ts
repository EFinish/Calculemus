// The JS↔Go boundary (DESIGN.md §7, §12): evaluate(universe) → verdicts,
// plus the on-demand revise(target) → retraction sets. Everything else in
// the app is rendering and editing.
import type { Revision, Universe, Verdicts } from "./types";

declare global {
  // Defined by /wasm_exec.js (classic script in index.html).
  const Go: new () => { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): Promise<void> };
  function calculemusEvaluate(universeJSON: string, scenario: string): string;
  function calculemusRevise(
    universeJSON: string,
    scenario: string,
    target: string,
    wantTrue: boolean,
  ): string;
}

let booting: Promise<void> | null = null;

function boot(): Promise<void> {
  booting ??= (async () => {
    const go = new Go();
    const { instance } = await WebAssembly.instantiateStreaming(
      fetch("/calculemus.wasm"),
      go.importObject,
    );
    void go.run(instance); // resolves only if the Go program exits; never awaited
    // main() registers the global asynchronously — wait for it to appear.
    for (let i = 0; !(globalThis as { calculemusEvaluate?: unknown }).calculemusEvaluate; i++) {
      if (i > 500) throw new Error("engine failed to register calculemusEvaluate");
      await new Promise((r) => setTimeout(r, 10));
    }
  })();
  return booting;
}

export async function evaluate(universe: Universe, scenario = ""): Promise<Verdicts> {
  await boot();
  const result = JSON.parse(calculemusEvaluate(JSON.stringify(universe), scenario)) as
    | Verdicts
    | { error: string };
  if ("error" in result) throw new Error(result.error);
  return result;
}

export async function revise(
  universe: Universe,
  scenario: string,
  target: string,
  wantTrue: boolean,
): Promise<Revision> {
  await boot();
  const result = JSON.parse(
    calculemusRevise(JSON.stringify(universe), scenario, target, wantTrue),
  ) as Revision | { error: string };
  if ("error" in result) throw new Error(result.error);
  return result;
}
