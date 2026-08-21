package core

import (
	"fmt"
	"slices"
)

type EdgeType string

const (
	// EdgeShares: two formulas mention at least one common atom (structural).
	EdgeShares EdgeType = "shares"
	// EdgeContradicts: both assertions belong to a minimal unsatisfiable core.
	EdgeContradicts EdgeType = "contradicts"
	// EdgeChains: the From argument's conclusion entails a premise of the To
	// argument — arguments composing into larger proofs.
	EdgeChains EdgeType = "chains"
	// EdgeEntails: the From statement/formula alone forces the To one (§5
	// as-built). Trivial sources (self-contradictions) and targets
	// (tautologies) are skipped, equivalents collapsed, and the set
	// transitively reduced — the Hasse skeleton, not the closure.
	EdgeEntails EdgeType = "entails"
)

type Edge struct {
	Type EdgeType `json:"type"`
	From string   `json:"from"`
	To   string   `json:"to"`
}

type ArgumentVerdict struct {
	ID    string `json:"id"`
	Valid bool   `json:"valid"`
	// Form names a recognized argument shape ("modus ponens", "Barbara", …).
	// Purely decorative (guardrail R1): set only on valid arguments, and an
	// unrecognized valid argument is simply unlabeled.
	Form string `json:"form,omitempty"`
	// Countermodel, present when invalid: an assignment of atoms under which
	// every premise holds and the conclusion fails.
	Countermodel map[string]bool `json:"countermodel,omitempty"`
}

// Verdicts is everything the engine derives from a universe. It is the whole
// product surface of core: consistency, diagnosis, forced truths, argument
// validity, and the derived edges of the web.
type Verdicts struct {
	Consistent bool `json:"consistent"`
	// UnsatCore, present when inconsistent: a minimal set of active
	// assertions (formula refs) that cannot coexist — remove any one and the
	// universe is consistent again.
	UnsatCore []string `json:"unsatCore,omitempty"`
	// EntailedTrue / EntailedFalse: statement ids the active assertions
	// force. Empty when the universe is inconsistent — classically a
	// contradiction entails everything, so deriving would be noise; the
	// diagnosis (UnsatCore) replaces it (DESIGN.md §4.1, the explosion guard).
	EntailedTrue  []string `json:"entailedTrue,omitempty"`
	EntailedFalse []string `json:"entailedFalse,omitempty"`
	// Vacuous: IMPLIES formulas whose antecedent the assertions force false —
	// true, but only vacuously (DESIGN.md §4.2). Empty when inconsistent.
	Vacuous []string `json:"vacuous,omitempty"`
	// Discoveries: atomic statements over the existing vocabulary that the
	// assertions force but nobody authored — proposals, adopted only by the
	// user (id and text are left empty for the app to fill). Suppressed when
	// inconsistent (§4.1).
	Discoveries []Statement       `json:"discoveries,omitempty"`
	Arguments   []ArgumentVerdict `json:"arguments,omitempty"`
	Edges       []Edge            `json:"edges,omitempty"`
	// BoundedDomain, when set, means relational (M6) semantics were in play:
	// verdicts were checked over worlds with at most this many things.
	// Countermodels found are absolute; "valid" is valid-up-to-this-bound.
	BoundedDomain int `json:"boundedDomain,omitempty"`
}

// Evaluate runs every query against the universe's active assertions.
// This is the single intended boundary for callers (and, at M1, for the
// WASM bridge): universe in, verdicts out.
func Evaluate(u *Universe) (*Verdicts, error) {
	return evaluate(u, nil)
}

// EvaluateScenario evaluates under a named scenario's assertion toggles.
func EvaluateScenario(u *Universe, name string) (*Verdicts, error) {
	for _, sc := range u.Scenarios {
		if sc.Name == name {
			return evaluate(u, sc.Toggles)
		}
	}
	return nil, fmt.Errorf("unknown scenario %q", name)
}

func evaluate(u *Universe, toggles map[string]bool) (*Verdicts, error) {
	ix, err := buildIndex(u)
	if err != nil {
		return nil, err
	}
	c, err := compile(ix)
	if err != nil {
		return nil, err
	}
	active := activeAssertions(u, toggles)
	assume := make([]int, len(active))
	for i, ref := range active {
		assume[i] = c.lit(ref)
	}

	v := &Verdicts{}
	if c.grounded != nil {
		v.BoundedDomain = c.grounded.domain
	}
	var firstModel []bool
	v.Consistent, firstModel = c.sat(assume...)

	if v.Consistent {
		e := newEntailer(c, assume, firstModel)
		for _, s := range u.Statements {
			if e.forcedTrue(c.lit(s.ID)) {
				v.EntailedTrue = append(v.EntailedTrue, s.ID)
			} else if e.forcedFalse(c.lit(s.ID)) {
				v.EntailedFalse = append(v.EntailedFalse, s.ID)
			}
		}
		for i := range u.Formulas {
			f := &u.Formulas[i]
			if f.Op != OpImplies {
				continue
			}
			// Antecedent forced false ⇒ the conditional holds vacuously.
			if e.forcedFalse(c.lit(f.Args[0])) {
				v.Vacuous = append(v.Vacuous, f.ID)
			}
		}
		v.Discoveries = computeDiscoveries(u, active)
	} else {
		v.UnsatCore = minimizeCore(c, active)
		for i := 0; i < len(v.UnsatCore); i++ {
			for j := i + 1; j < len(v.UnsatCore); j++ {
				v.Edges = append(v.Edges, Edge{EdgeContradicts, v.UnsatCore[i], v.UnsatCore[j]})
			}
		}
	}

	// Argument validity is per-argument and independent of assertions, so it
	// keeps working even when the universe is inconsistent.
	for _, arg := range u.Arguments {
		lits := make([]int, 0, len(arg.Premises)+1)
		for _, p := range arg.Premises {
			lits = append(lits, c.lit(p))
		}
		lits = append(lits, -c.lit(arg.Conclusion))
		ok, model := c.sat(lits...)
		av := ArgumentVerdict{ID: arg.ID, Valid: !ok}
		if av.Valid {
			av.Form = classifyForm(ix, &arg)
		}
		if ok {
			av.Countermodel = make(map[string]bool, len(u.Statements))
			for _, s := range u.Statements {
				av.Countermodel[s.ID] = model[c.lit(s.ID)]
			}
		}
		v.Arguments = append(v.Arguments, av)
	}

	v.Edges = append(v.Edges, sharesEdges(ix)...)
	v.Edges = append(v.Edges, chainsEdges(u, c)...)
	v.Edges = append(v.Edges, entailsEdges(u, c)...)
	return v, nil
}

// entailsEdges computes pairwise entailment between authored statements and
// formulas: X ⊨ Y iff (X ∧ ¬Y) is unsatisfiable. Assertion-independent, like
// argument validity. A model pool prunes most pairs without solver calls: any
// model where X holds and Y fails is a witness that X ⊭ Y.
func entailsEdges(u *Universe, c *compiled) []Edge {
	var ids []string
	for _, s := range u.Statements {
		ids = append(ids, s.ID)
	}
	for _, f := range u.Formulas {
		ids = append(ids, f.ID)
	}
	n := len(ids)
	if n < 2 {
		return nil
	}

	var pool [][]bool
	refuted := func(x, y int) bool { // pool model with x true, y false?
		for _, m := range pool {
			if m[x] && !m[y] {
				return true
			}
		}
		return false
	}

	// Trivial nodes say nothing: an unsatisfiable X entails everything, a
	// tautological Y is entailed by everything.
	lit := make([]int, n)
	ok := make([]bool, n)
	for i, id := range ids {
		lit[i] = c.lit(id)
		if sat, m := c.sat(lit[i]); sat {
			if sat2, m2 := c.sat(-lit[i]); sat2 {
				ok[i] = true
				pool = append(pool, m, m2)
			}
		}
	}

	ent := make([][]bool, n)
	for i := range ent {
		ent[i] = make([]bool, n)
		if !ok[i] {
			continue
		}
		for j := range ent[i] {
			if i == j || !ok[j] || refuted(lit[i], lit[j]) {
				continue
			}
			sat, m := c.sat(lit[i], -lit[j])
			if sat {
				pool = append(pool, m)
			} else {
				ent[i][j] = true
			}
		}
	}

	// Collapse mutual entailment to one representative (first in id order) so
	// equivalent items don't spray edges; then transitively reduce.
	rep := make([]int, n)
	for i := range rep {
		rep[i] = i
		for j := 0; j < i; j++ {
			if ent[i][j] && ent[j][i] {
				rep[i] = rep[j]
				break
			}
		}
	}
	var edges []Edge
	for i := 0; i < n; i++ {
		if rep[i] != i {
			continue
		}
		for j := 0; j < n; j++ {
			if rep[j] != j || !ent[i][j] {
				continue
			}
			redundant := false
			for k := 0; k < n && !redundant; k++ {
				redundant = rep[k] == k && k != i && k != j && ent[i][k] && ent[k][j]
			}
			if !redundant {
				edges = append(edges, Edge{EdgeEntails, ids[i], ids[j]})
			}
		}
	}
	return edges
}

// activeAssertions resolves toggles over the assertion list and returns the
// deduplicated refs held true. A toggle overrides matching assertions; a
// true toggle with no matching assertion asserts the ref for this run.
func activeAssertions(u *Universe, toggles map[string]bool) []string {
	seen := map[string]bool{}
	var out []string
	add := func(ref string) {
		if !seen[ref] {
			seen[ref] = true
			out = append(out, ref)
		}
	}
	toggled := map[string]bool{}
	for _, a := range u.Assertions {
		active := a.Active
		if t, ok := toggles[a.Formula]; ok {
			active = t
			toggled[a.Formula] = true
		}
		if active {
			add(a.Formula)
		}
	}
	// Deterministic order for toggle-only assertions: universe id order.
	for _, s := range u.Statements {
		if toggles[s.ID] && !toggled[s.ID] {
			add(s.ID)
		}
	}
	for _, f := range u.Formulas {
		if toggles[f.ID] && !toggled[f.ID] {
			add(f.ID)
		}
	}
	return out
}

// minimizeCore shrinks an unsatisfiable assertion set to a minimal one by
// deletion: drop each member in turn and keep the drop whenever the rest is
// still unsatisfiable. Every remaining member is then necessary. The hard
// literals are never candidates for deletion (revision holds its target
// fixed while shrinking the assertions around it).
func minimizeCore(c *compiled, active []string, hard ...int) []string {
	core := slices.Clone(active)
	for i := 0; i < len(core); {
		trial := slices.Concat(core[:i], core[i+1:])
		lits := slices.Clone(hard)
		for _, ref := range trial {
			lits = append(lits, c.lit(ref))
		}
		if ok, _ := c.sat(lits...); !ok {
			core = trial
		} else {
			i++
		}
	}
	return core
}

func sharesEdges(ix *index) []Edge {
	var edges []Edge
	fs := ix.u.Formulas
	atomSets := make([]map[string]bool, len(fs))
	for i := range fs {
		atomSets[i] = ix.atoms(fs[i].ID)
	}
	for i := range fs {
		for j := i + 1; j < len(fs); j++ {
			for atom := range atomSets[i] {
				if atomSets[j][atom] {
					edges = append(edges, Edge{EdgeShares, fs[i].ID, fs[j].ID})
					break
				}
			}
		}
	}
	return edges
}

func chainsEdges(u *Universe, c *compiled) []Edge {
	var edges []Edge
	for i := range u.Arguments {
		for j := range u.Arguments {
			if i == j {
				continue
			}
			a, b := &u.Arguments[i], &u.Arguments[j]
			for _, p := range b.Premises {
				// a.Conclusion ⊨ p iff (conclusion ∧ ¬p) is unsatisfiable.
				if ok, _ := c.sat(c.lit(a.Conclusion), -c.lit(p)); !ok {
					edges = append(edges, Edge{EdgeChains, a.ID, b.ID})
					break
				}
			}
		}
	}
	return edges
}
