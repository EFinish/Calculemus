# Calculemus — agent instructions

The design source of truth is `../DESIGN.md` (workspace root). Read §7
(architecture), §8 (current milestone), and §9 (guardrails) before changing
anything structural.

## Non-negotiable guardrails (from DESIGN.md §9)

- `core/` stays zero-dependency and never imports from `wasm/`, `app/`, or
  `server/`. All logic lives in `core/`; the app renders and edits, never
  reasons.
- The engine is **semantic**: features are built on satisfiability queries,
  never on recognizing named argument forms (those are decorative
  annotations only).
- No server, database, Docker, protobuf, or deployment work before
  milestone M5.
- Refactor under tests; never rewrite. `core/oracle_test.go` property-checks
  the solver pipeline against a brute-force truth-table oracle — any solver
  or encoding change must keep those tests passing unmodified.
- The JS↔Go boundary (M1) is exactly one call:
  `evaluate(universeJSON) → verdictsJSON`.

## Conventions

- Verify with `go test ./...` (fast; runs 500 oracle-checked random
  universes).
- The universe JSON document is the persistence format — schema changes bump
  `Universe.Version` and need a migration note in DESIGN.md.
- Explosion guard: when a universe is inconsistent, entailment/vacuous
  results are suspended and diagnosis (unsat core) replaces them
  (DESIGN.md §4.1). Don't "fix" empty EntailedTrue on inconsistent input —
  it's deliberate.
