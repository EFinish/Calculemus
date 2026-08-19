#!/usr/bin/env bash
# Run Calculemus: verify the engine, then evaluate the example universe —
# once as authored (consistent) and once under the contradictory scenario.
set -euo pipefail
cd "$(dirname "$0")"

echo "── go test ./... ──────────────────────────────────────────"
go test ./...

echo
echo "── examples/ball.json ─────────────────────────────────────"
go run ./cmd/calculemus examples/ball.json

echo
echo "── examples/ball.json  -scenario 'blue too' ───────────────"
# Contradictory on purpose; the CLI exits 1 on inconsistency, which is the
# demo's point — don't let it stop the script.
go run ./cmd/calculemus -scenario "blue too" examples/ball.json || true

# Evaluate your own universe:  ./run.sh path/to/universe.json [scenario]
if [[ $# -ge 1 ]]; then
  echo
  echo "── $1 ${2:+ -scenario \"$2\"} ──"
  go run ./cmd/calculemus ${2:+-scenario "$2"} "$1" || true
fi
