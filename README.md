# Calculemus

> *"…there would be no more need of disputation between two philosophers than
> between two accountants. It would suffice to take their pencils in hand and
> say to each other: **calculemus** — let us calculate."*
> — Gottfried Wilhelm Leibniz

A workbench for building logical universes. You compose statements from
atomic pieces, chain them into arguments, assert what you take to be true —
and the engine computes the rest: whether your universe is consistent,
whether your arguments are valid, which conclusions are forced, and (when you
contradict yourself) exactly which assertions are the fight.

The full product design, including the archaeology of four prior versions and
the guardrails written against their causes of death, lives in
[`../DESIGN.md`](../DESIGN.md).

## Status

**M3 — the canvas.** The universe as a visible web: statements, formulas,
and arguments as nodes; every edge derived by the engine (shares /
contradicts / chains) and color-coded, with per-type filters. Truth state
is encoded in the node itself; clicking a node selects it in the
inspector. Drag to arrange — positions persist in the universe document.

**M2 — verdicts & inspector.** Click any statement, formula, or argument
to inspect it: truth states explained, vacuous conditionals explained,
invalid arguments shown with their countermodel (a world where every
premise holds and the conclusion fails), chains listed. The contradiction
diagnosis is actionable — unassert a core member from inside the verdict
pane.

**M1 — the app.** `wasm/` bridges the engine to the browser (one call:
`evaluate(universeJSON) → verdictsJSON`); `app/` is the Vue 3 workbench —
guided composer, live verdicts, localStorage autosave, JSON export/import.
`make e2e` runs the M1 dogfood test in headless Firefox: build a universe,
reload the tab, everything survives.

**M0 — the engine.** `core/` is a zero-dependency Go package:

- the universe data model (statements → formulas → assertions/arguments),
  JSON-serializable as one document
- a Tseitin compiler + small DPLL solver
- the four semantic queries, all built on one primitive (satisfiability):
  consistency, argument validity (with countermodels), entailed truths, and
  minimal-unsat-core diagnosis — plus derived web edges (shares,
  contradicts, chains)
- an exhaustive test suite, property-checked against a brute-force
  truth-table oracle

## Run it

```bash
./run.sh                       # tests, then the example universe two ways:
                               # as authored (consistent, derives conclusions)
                               # and under the "blue too" scenario (contradictory,
                               # prints the minimal conflicting assertion set)

./run.sh my-universe.json      # also evaluate your own universe
go test ./...                  # just the engine verification
```

Or drive the CLI directly:

```bash
go run ./cmd/calculemus examples/ball.json
go run ./cmd/calculemus -scenario "blue ball world" examples/ball.json
go run ./cmd/calculemus -json examples/ball.json   # raw verdicts JSON
```

`examples/ball.json` doubles as documentation of the universe document
format. The CLI exits 1 when the universe is contradictory, 2 on bad input.

## Design in one line

**The user authors structure; the machine derives relations.** Every edge in
the web is a theorem, never a drawing. Validity is decided by semantics
(no countermodel exists), never by pattern-matching named argument forms.

## Layout (per DESIGN.md §7)

```
core/    the engine — model, solver, queries          [M0, here]
wasm/    GOOS=js bridge: evaluate(universe) → verdicts [M1]
app/     Vue 3 + vue-flow frontend                     [M1–M3]
server/  thin Go document store, imports core/         [M5 — not before]
```
