// Smoke test for the WASM bridge: load the compiled engine in Node, call
// calculemusEvaluate on the example universe, check the derived verdicts.
// Run via `make smoke`.
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

// wasm_exec.js is a classic script defining globalThis.Go.
const shim = await readFile(join(root, "app/public/wasm_exec.js"), "utf8");
new Function(shim)();

const go = new globalThis.Go();
const wasm = await readFile(join(root, "app/public/calculemus.wasm"));
const { instance } = await WebAssembly.instantiate(wasm, go.importObject);
go.run(instance); // resolves only when the Go program exits — don't await

// main() sets the global asynchronously; wait for it.
for (let i = 0; !globalThis.calculemusEvaluate; i++) {
  if (i > 500) throw new Error("calculemusEvaluate never appeared");
  await new Promise((r) => setTimeout(r, 10));
}

const universe = await readFile(join(root, "examples/ball.json"), "utf8");

const assert = (cond, msg) => {
  if (!cond) {
    console.error("FAIL:", msg);
    process.exit(1);
  }
};

const v = JSON.parse(globalThis.calculemusEvaluate(universe, ""));
assert(!v.error, `unexpected error: ${v.error}`);
assert(v.consistent === true, "ball universe should be consistent");
assert(v.entailedTrue.includes("s_play"), "s_play should be entailed true");
assert(v.entailedFalse.includes("s_blue"), "s_blue should be entailed false");

const blue = JSON.parse(globalThis.calculemusEvaluate(universe, "blue too"));
assert(blue.consistent === false, "'blue too' scenario should be contradictory");
assert(blue.unsatCore.length === 4, "core should have 4 members");

const bad = JSON.parse(globalThis.calculemusEvaluate("{not json", ""));
assert(typeof bad.error === "string", "bad input should return an error field");

console.log("wasm bridge smoke test: OK");
process.exit(0);
