package core

import (
	"slices"
	"testing"
)

func entailEdges(t *testing.T, u *Universe) []Edge {
	t.Helper()
	v, err := Evaluate(u)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	var out []Edge
	for _, e := range v.Edges {
		if e.Type == EdgeEntails {
			out = append(out, e)
		}
	}
	return out
}

// The diamond: AND(p,q) entails each conjunct, each conjunct entails
// OR(p,q). Transitive reduction must drop AND→OR (reachable via p).
func TestEntailsEdgesDiamond(t *testing.T) {
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			{ID: "p", Text: "p"},
			{ID: "q", Text: "q"},
		},
		Formulas: []Formula{
			{ID: "both", Op: OpAnd, Args: []string{"p", "q"}},
			{ID: "either", Op: OpOr, Args: []string{"p", "q"}},
		},
	}
	got := entailEdges(t, u)
	want := []Edge{
		{EdgeEntails, "both", "p"},
		{EdgeEntails, "both", "q"},
		{EdgeEntails, "p", "either"},
		{EdgeEntails, "q", "either"},
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("missing edge %v", w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("edges = %v, want exactly %v (transitive reduction)", got, want)
	}
}

// Equivalent items collapse to one representative: NOT(NOT p) ≡ p, so the
// double negation gets no edges of its own and no p↔¬¬p pair appears.
func TestEntailsEdgesCollapseEquivalents(t *testing.T) {
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			{ID: "p", Text: "p"},
			{ID: "q", Text: "q"},
		},
		Formulas: []Formula{
			{ID: "np", Op: OpNot, Args: []string{"p"}},
			{ID: "nnp", Op: OpNot, Args: []string{"np"}},
			{ID: "either", Op: OpOr, Args: []string{"p", "q"}},
		},
	}
	got := entailEdges(t, u)
	for _, e := range got {
		if e.From == "nnp" || e.To == "nnp" {
			t.Errorf("nnp ≡ p must be collapsed, got edge %v", e)
		}
	}
	if !slices.Contains(got, Edge{EdgeEntails, "p", "either"}) {
		t.Errorf("representative p must keep its edges, got %v", got)
	}
}

// Contradictory sources and tautological targets are noise, not theorems.
func TestEntailsEdgesSkipTrivial(t *testing.T) {
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			{ID: "p", Text: "p"},
			{ID: "q", Text: "q"},
		},
		Formulas: []Formula{
			{ID: "np", Op: OpNot, Args: []string{"p"}},
			{ID: "bang", Op: OpAnd, Args: []string{"p", "np"}}, // unsatisfiable
			{ID: "taut", Op: OpOr, Args: []string{"p", "np"}},  // tautology
		},
	}
	for _, e := range entailEdges(t, u) {
		if e.From == "bang" || e.To == "taut" {
			t.Errorf("trivial entailment leaked: %v", e)
		}
	}
}

// Term semantics flow through: "all greeks are men" ⊨ "no greeks are men"?
// Never — but A and its O-negation are mutually exclusive, not entailing.
// What does hold: E("greeks","gods") entails E-converse via symmetric truth…
// kept simple: A-form entails nothing alone under the Boolean reading, so a
// two-statement categorical universe has no entails edges.
func TestEntailsEdgesBooleanReading(t *testing.T) {
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			stmt("a", "greeks", QuantAll, "men", QualIs),
			stmt("i", "greeks", QuantSome, "men", QualIs),
		},
	}
	if got := entailEdges(t, u); len(got) != 0 {
		t.Errorf("A must not entail I without existential import, got %v", got)
	}
}
