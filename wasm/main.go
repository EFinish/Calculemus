//go:build js && wasm

// The WASM bridge: two calls (DESIGN.md §7, §12).
//
//	calculemusEvaluate(universeJSON, scenarioName) → verdictsJSON
//	calculemusRevise(universeJSON, scenarioName, targetRef, wantTrue) → revisionJSON
//
// scenarioName "" means the universe's base assertions. Revision is separate
// from Evaluate because it is per-target and on-demand (§12); Evaluate stays
// the one recurring call. Errors come back as {"error": "..."} — the bridge
// never throws across the boundary.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/EFinish/Calculemus/core"
)

func main() {
	js.Global().Set("calculemusEvaluate", js.FuncOf(evaluate))
	js.Global().Set("calculemusRevise", js.FuncOf(revise))
	select {} // keep the Go runtime alive for the page's lifetime
}

func revise(_ js.Value, args []js.Value) any {
	respond := func(v any) any {
		b, err := json.Marshal(v)
		if err != nil {
			return `{"error":"marshal failure"}`
		}
		return string(b)
	}
	if len(args) < 4 {
		return respond(map[string]string{"error": "calculemusRevise needs (universe, scenario, target, wantTrue)"})
	}
	var u core.Universe
	if err := json.Unmarshal([]byte(args[0].String()), &u); err != nil {
		return respond(map[string]string{"error": "invalid universe JSON: " + err.Error()})
	}
	r, err := core.Revise(&u, args[1].String(), args[2].String(), args[3].Bool())
	if err != nil {
		return respond(map[string]string{"error": err.Error()})
	}
	return respond(r)
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
