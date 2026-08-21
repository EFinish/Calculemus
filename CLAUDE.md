# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Calculemus is a workbench for building logical universes: statements compose
into formulas and arguments, the user asserts what they hold true, and a SAT
engine derives everything else (consistency, validity, entailed truths,
diagnosis, discoveries). The design source of truth is `../DESIGN.md` in the
workspace root — §7 (architecture), §9 (guardrails), §10 (epistemological
stance), §11 (relations) before changing anything structural.

## Commands

```bash
go test ./...                          # engine + server tests (~1s; includes ~1000 oracle-checked random universes)
go test ./core/ -run TestBarbara       # one Go test
go test ./core/ -run XXX -bench EvaluateChildBall   # engine benchmark
make wasm                              # build engine → app*/public/ (REQUIRED after any core/ change)
make smoke                             # wasm + Node smoke test of the bridge
make dev                               # wasm + vite dev server (app at localhost:5173)
make e2e                               # wasm + full Playwright suite, headless Firefox
cd app && npx playwright test m6       # one e2e spec file (also: gallery, editing, discoveries, m1–m5)
cd app && npm run typecheck            # vue-tsc
make build                             # tests + wasm + both app builds
make serve                             # production: one Go binary serving both editions + /api on :8737
./run.sh                               # CLI demo; go run ./cmd/calculemus examples/<name>.json evaluates any universe
```

The e2e suite starts its own vite (:5199) and sharing server (:8737). If a
core change alters engine behavior, run `make wasm` before e2e or the app
tests against the stale binary.

## Architecture

One Go module, one Vue app, one frozen copy of the app, one thin server —
all reasoning lives in `core/` and reaches the browser as WASM.

- **`core/`** — zero dependencies, never imports from the other dirs. Flow:
  `model.go` (Universe document) → `validate.go` (index, refs, arities, DAG)
  → `cnf.go` `compile()` (Tseitin) → `solver.go` (DPLL, occurrence-list unit
  propagation) → `queries.go` `Evaluate()` (the ONE public boundary:
  universe in, Verdicts out). Term semantics has two modes, gated in
  `compile()`: pure-copular universes use the **Venn-region encoding**
  (`terms.go`, exact for monadic logic); any verb or individual switches the
  whole term layer to **bounded-domain grounding** (`relational.go`,
  `Verdicts.BoundedDomain` reports the bound — countermodels absolute,
  "valid" means valid-up-to-bound). `entailer.go` (model-pool pruning) and
  `discoveries.go` (saturation: propose entailed-but-unauthored statements)
  sit on top; `forms.go` is decorative labeling only.
- **`wasm/`** — the entire JS↔Go surface is one call:
  `calculemusEvaluate(universeJSON, scenario) → verdictsJSON`. Errors return
  as `{"error": ...}`, never thrown.
- **`app/`** — Vue 3 + vue-flow. `src/store.ts` is the heart: one reactive
  Universe, persisted to localStorage **synchronously on every change**
  (only engine evaluation is debounced — a debounced save loses the last
  change on reload); scenario-aware assert toggles; edit-in-place with
  stable ids. `src/engine.ts` wraps the WASM call; `src/phrase.ts` is the
  single source of sentence phrasing. The app renders and edits; it never
  reasons.
- **`app-boolean/`** — the pre-M6 workbench, FROZEN at `/boolean/`. Never
  evolve it alongside `app/`; it exists so users can go back. Differences it
  must keep: relative base (`./`), query-param share links (`?u=`), the
  original localStorage key `calculemus.universe` (the relational app uses
  `calculemus.v2.universe`).
- **`server/`** — stdlib-only document store for sharing: validates via
  `core.Validate`, never evaluates; immutable snapshots (re-share mints a
  new id, no update/delete); also serves both built apps. Keep it dumb.
- **`examples/`** — canonical universes, shared verbatim by the CLI and the
  app's Try-me gallery (Vite inlines them; dev needs `server.fs.allow`).

## Invariants that look like bugs but aren't

- **Explosion guard** (§4.1): when inconsistent, EntailedTrue/False,
  Vacuous, and Discoveries are deliberately empty; the unsat core replaces
  them. Argument validity keeps working (it's assertion-independent).
- **Boolean reading** (no existential import): Darapti is invalid, "all
  unicorns are white" is true with no unicorns. The unicorn example and
  tests pin this.
- **Verb/kind strings are identity**: "throw" and "throws" are different
  relations; kinds and individuals are separate namespaces even when
  spelled alike. Term matching is `termKey` (trim + lowercase).
- **Discoveries are proposals**: the machine never authors statements; the
  user adopts them (id/text left empty for the app to fill).

## Working rules

- **Refactor under tests; never rewrite.** The oracle property suites
  (`oracle_test.go`, `syllogism_test.go`, `relational_test.go`,
  `discoveries_test.go`) check the solver pipeline against brute-force
  enumeration; any solver or encoding change must keep them passing
  unmodified. They have already validated one full solver rewrite.
- Named argument forms decorate verdicts, never produce them (guardrail R1).
- One logic, declared (R7): classical, with the bounded-domain caveat
  surfaced in the UI. New semantics need a DESIGN.md section first.
- Universe JSON is the persistence format; schema changes must keep old
  documents parsing (M6 did this: `verb: ""` = copula) and get a note in
  DESIGN.md.
- e2e specs: use the `helpers.ts` verify-and-retry helpers (`setAssert`,
  `addArgument`, `inspect`, `openEdit`) instead of raw clicks — coordinate
  clicks race verdict re-renders; a mis-aimed premise click once produced a
  valid-but-wrong-shaped argument that only the form-badge assertion caught.
- After the work is verified, commit with a conventional-commits message;
  push only when asked.
