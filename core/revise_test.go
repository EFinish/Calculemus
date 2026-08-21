package core

import (
	"slices"
	"testing"
)

func reviseUniverse() *Universe {
	return &Universe{
		Version: 1,
		Statements: []Statement{
			{ID: "p", Text: "socrates is a man"},
			{ID: "q", Text: "socrates is mortal"},
		},
		Formulas: []Formula{
			{ID: "imp", Op: OpImplies, Args: []string{"p", "q"}},
		},
		Assertions: []Assertion{
			{Formula: "p", Active: true},
			{Formula: "imp", Active: true},
		},
	}
}

// q is forced true by {p, p→q}. Denying q costs exactly one of them.
func TestReviseOffersMinimalRetractions(t *testing.T) {
	r, err := Revise(reviseUniverse(), "", "q", false)
	if err != nil {
		t.Fatalf("Revise: %v", err)
	}
	if !r.Possible || r.AlreadySatisfiable {
		t.Fatalf("¬q is possible but not free: %+v", r)
	}
	want := [][]string{{"imp"}, {"p"}}
	if !slices.EqualFunc(r.Retractions, want, slices.Equal) {
		t.Errorf("Retractions = %v, want %v", r.Retractions, want)
	}
}

func TestReviseAlreadySatisfiable(t *testing.T) {
	r, err := Revise(reviseUniverse(), "", "q", true)
	if err != nil {
		t.Fatalf("Revise: %v", err)
	}
	if !r.Possible || !r.AlreadySatisfiable || len(r.Retractions) != 0 {
		t.Errorf("believing q costs nothing, got %+v", r)
	}
}

// A self-contradictory belief is beyond any retraction.
func TestReviseImpossibleBelief(t *testing.T) {
	u := reviseUniverse()
	u.Formulas = append(u.Formulas,
		Formula{ID: "np", Op: OpNot, Args: []string{"p"}},
		Formula{ID: "bang", Op: OpAnd, Args: []string{"p", "np"}},
	)
	r, err := Revise(u, "", "bang", true)
	if err != nil {
		t.Fatalf("Revise: %v", err)
	}
	if r.Possible {
		t.Errorf("p ∧ ¬p must be impossible, got %+v", r)
	}
}

// Independent chains: {a, a→x} and {b, b→x} both force x; denying x must
// break BOTH chains, so every retraction set has size two.
func TestReviseCrossChainRetractions(t *testing.T) {
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			{ID: "a", Text: "a"}, {ID: "b", Text: "b"}, {ID: "x", Text: "x"},
		},
		Formulas: []Formula{
			{ID: "ax", Op: OpImplies, Args: []string{"a", "x"}},
			{ID: "bx", Op: OpImplies, Args: []string{"b", "x"}},
		},
		Assertions: []Assertion{
			{Formula: "a", Active: true}, {Formula: "ax", Active: true},
			{Formula: "b", Active: true}, {Formula: "bx", Active: true},
		},
	}
	r, err := Revise(u, "", "x", false)
	if err != nil {
		t.Fatalf("Revise: %v", err)
	}
	if len(r.Retractions) != 4 {
		t.Fatalf("want the 4 cross-chain pairs, got %v", r.Retractions)
	}
	for _, set := range r.Retractions {
		if len(set) != 2 {
			t.Errorf("every retraction must break both chains: %v", set)
		}
		fromFirst := slices.Contains(set, "a") || slices.Contains(set, "ax")
		fromSecond := slices.Contains(set, "b") || slices.Contains(set, "bx")
		if !fromFirst || !fromSecond {
			t.Errorf("set %v misses a chain", set)
		}
	}
}

// Retracting toward a target that is itself asserted: the assertion of the
// target never appears in its own retraction sets.
func TestReviseNeverRetractsSelf(t *testing.T) {
	u := reviseUniverse()
	u.Formulas = append(u.Formulas, Formula{ID: "nq", Op: OpNot, Args: []string{"q"}})
	u.Assertions = append(u.Assertions, Assertion{Formula: "nq", Active: true})
	// Universe now inconsistent-ish? p, p→q force q; nq denies it. Believing
	// nq true: retract from {p, imp}, never nq itself.
	r, err := Revise(u, "", "nq", true)
	if err != nil {
		t.Fatalf("Revise: %v", err)
	}
	for _, set := range r.Retractions {
		if slices.Contains(set, "nq") {
			t.Errorf("a belief must not retract itself: %v", set)
		}
	}
}

func TestReviseUnknownTargetAndScenario(t *testing.T) {
	if _, err := Revise(reviseUniverse(), "", "ghost", true); err == nil {
		t.Error("unknown target must error")
	}
	if _, err := Revise(reviseUniverse(), "ghost", "q", true); err == nil {
		t.Error("unknown scenario must error")
	}
}

// Scenario toggles change the assertion base the retractions come from.
func TestReviseUnderScenario(t *testing.T) {
	u := reviseUniverse()
	u.Scenarios = []Scenario{{Name: "doubt", Toggles: map[string]bool{"p": false}}}
	r, err := Revise(u, "doubt", "q", false)
	if err != nil {
		t.Fatalf("Revise: %v", err)
	}
	// With p off, only imp remains active and ¬q is already fine.
	if !r.AlreadySatisfiable {
		t.Errorf("with p toggled off, ¬q needs no retraction: %+v", r)
	}
}
