//go:build js && wasm

// The WASM bridge: the entire JS↔Go boundary is one call (DESIGN.md §7).
//
//	calculemusEvaluate(universeJSON, scenarioName) → verdictsJSON
//
// scenarioName "" evaluates the universe's active assertions. Errors come
// back as {"error": "..."} — the bridge never throws across the boundary.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/EFinish/Calculemus/core"
)

func main() {
	js.Global().Set("calculemusEvaluate", js.FuncOf(evaluate))
	select {} // keep the Go runtime alive for the page's lifetime
}

func evaluate(_ js.Value, args []js.Value) any {
	respond := func(v any) any {
		b, err := json.Marshal(v)
		if err != nil {
			return `{"error":"marshal failure"}`
		}
		return string(b)
	}
	if len(args) < 1 {
		return respond(map[string]string{"error": "missing universe JSON"})
	}
	var u core.Universe
	if err := json.Unmarshal([]byte(args[0].String()), &u); err != nil {
		return respond(map[string]string{"error": "invalid universe JSON: " + err.Error()})
	}
	scenario := ""
	if len(args) >= 2 {
		scenario = args[1].String()
	}

	var verdicts *core.Verdicts
	var err error
	if scenario != "" {
		verdicts, err = core.EvaluateScenario(&u, scenario)
	} else {
		verdicts, err = core.Evaluate(&u)
	}
	if err != nil {
		return respond(map[string]string{"error": err.Error()})
	}
	return respond(verdicts)
}
