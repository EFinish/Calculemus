package core

import (
	"encoding/json"
	"slices"
	"testing"
)

// The M0 dogfood test (DESIGN.md §8): apriorio's README example, run
// forward. Assert "red→play", "blue→¬play", "red" — the engine must derive
// "play" (modus ponens) and "¬blue" (modus tollens) without either rule
// being written anywhere.
func ballUniverse() *Universe {
	return &Universe{
		Version: 1,
		Title:   "Is it time to play?",
		Statements: []Statement{
			{ID: "s_red", Text: "all of the ball is red", Subject: "the ball", Quantifier: QuantAll, Predicate: "red", Qualifier: QualIs},
			{ID: "s_blue", Text: "all of the ball is blue", Subject: "the ball", Quantifier: QuantAll, Predicate: "blue", Qualifier: QualIs},
			{ID: "s_play", Text: "all of the time to play is now", Subject: "the time to play", Quantifier: QuantAll, Predicate: "now", Qualifier: QualIs},
		},
		Formulas: []Formula{
			{ID: "not_play", Op: OpNot, Args: []string{"s_play"}},
			{ID: "f_red_play", Op: OpImplies, Args: []string{"s_red", "s_play"}},
			{ID: "f_blue_notplay", Op: OpImplies, Args: []string{"s_blue", "not_play"}},
			{ID: "f_or", Op: OpOr, Args: []string{"s_play", "s_blue"}},
		},
		Assertions: []Assertion{
			{Formula: "f_red_play", Active: true, Source: "hand"},
			{Formula: "f_blue_notplay", Active: true, Source: "hand"},
			{Formula: "s_red", Active: true, Source: "hand"},
		},
		Arguments: []Argument{
			{ID: "a_play", Title: "Is it time to play?", Premises: []string{"f_red_play", "s_red"}, Conclusion: "s_play"},
			{ID: "a_affirm", Title: "Affirming the consequent", Premises: []string{"f_red_play", "s_play"}, Conclusion: "s_red"},
			{ID: "a_addition", Title: "Addition", Premises: []string{"s_play"}, Conclusion: "f_or"},
		},
		Scenarios: []Scenario{
			{Name: "blue too", Toggles: map[string]bool{"s_blue": true}},
			{Name: "blue ball world", Toggles: map[string]bool{"s_red": false, "s_blue": true}},
		},
	}
}

func TestBallUniverse(t *testing.T) {
	u := ballUniverse()
	v, err := Evaluate(u)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !v.Consistent {
		t.Fatal("universe should be consistent")
	}
	wantTrue := []string{"s_red", "s_play"}
	if !slices.Equal(v.EntailedTrue, wantTrue) {
		t.Errorf("EntailedTrue = %v, want %v", v.EntailedTrue, wantTrue)
	}
	wantFalse := []string{"s_blue"}
	if !slices.Equal(v.EntailedFalse, wantFalse) {
		t.Errorf("EntailedFalse = %v, want %v", v.EntailedFalse, wantFalse)
	}
	// blue→¬play holds only because blue is forced false: vacuous (§4.2).
	if !slices.Equal(v.Vacuous, []string{"f_blue_notplay"}) {
		t.Errorf("Vacuous = %v, want [f_blue_notplay]", v.Vacuous)
	}
}

func TestBallArguments(t *testing.T) {
	u := ballUniverse()
	v, err := Evaluate(u)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	verdicts := map[string]ArgumentVerdict{}
	for _, av := range v.Arguments {
		verdicts[av.ID] = av
	}
	if !verdicts["a_play"].Valid {
		t.Error("modus ponens argument should be valid")
	}
	if !verdicts["a_addition"].Valid {
		t.Error("addition argument should be valid")
	}
	affirm := verdicts["a_affirm"]
	if affirm.Valid {
		t.Fatal("affirming the consequent should be invalid")
	}
	// The countermodel must actually witness the invalidity.
	cm := affirm.Countermodel
	if !cm["s_play"] || cm["s_red"] {
		t.Errorf("countermodel %v should have s_play=true, s_red=false", cm)
	}
}

func TestBallContradictionScenario(t *testing.T) {
	u := ballUniverse()
	v, err := EvaluateScenario(u, "blue too")
	if err != nil {
		t.Fatalf("EvaluateScenario: %v", err)
	}
	if v.Consistent {
		t.Fatal("asserting red and blue together should be contradictory")
	}
	wantCore := []string{"f_red_play", "f_blue_notplay", "s_red", "s_blue"}
	gotCore := slices.Clone(v.UnsatCore)
	slices.Sort(gotCore)
	slices.Sort(wantCore)
	if !slices.Equal(gotCore, wantCore) {
		t.Errorf("UnsatCore = %v, want %v", v.UnsatCore, wantCore)
	}
	// Explosion guard (§4.1): no derivation from a contradictory universe.
	if len(v.EntailedTrue) != 0 || len(v.EntailedFalse) != 0 || len(v.Vacuous) != 0 {
		t.Errorf("inconsistent universe must suspend entailment; got true=%v false=%v vacuous=%v",
			v.EntailedTrue, v.EntailedFalse, v.Vacuous)
	}
	// Contradicts edges: a clique over the 4-member core.
	contradicts := 0
	for _, e := range v.Edges {
		if e.Type == EdgeContradicts {
			contradicts++
		}
	}
	if contradicts != 6 {
		t.Errorf("want 6 contradicts edges over a 4-member core, got %d", contradicts)
	}
	// Validity is independent of assertions and keeps working.
	for _, av := range v.Arguments {
		if av.ID == "a_play" && !av.Valid {
			t.Error("a_play must stay valid in a contradictory universe")
		}
	}
}

func TestBallBlueWorldScenario(t *testing.T) {
	u := ballUniverse()
	v, err := EvaluateScenario(u, "blue ball world")
	if err != nil {
		t.Fatalf("EvaluateScenario: %v", err)
	}
	if !v.Consistent {
		t.Fatal("blue ball world should be consistent")
	}
	if !slices.Contains(v.EntailedTrue, "s_blue") {
		t.Errorf("EntailedTrue = %v, want s_blue included", v.EntailedTrue)
	}
	if !slices.Contains(v.EntailedFalse, "s_play") {
		t.Errorf("EntailedFalse = %v, want s_play included (blue → not play)", v.EntailedFalse)
	}
}

func TestBallEdges(t *testing.T) {
	u := ballUniverse()
	v, err := Evaluate(u)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	shares, chains := 0, 0
	var chainEdges []Edge
	for _, e := range v.Edges {
		switch e.Type {
		case EdgeShares:
			shares++
		case EdgeChains:
			chains++
			chainEdges = append(chainEdges, e)
		}
	}
	// All 4 formulas mention s_play, so every pair shares: C(4,2) = 6.
	if shares != 6 {
		t.Errorf("want 6 shares edges, got %d", shares)
	}
	// a_play concludes s_play, which is a premise of a_addition — a chain
	// nobody drew. (a_affirm also has premise s_play, hence 2 chains from
	// a_play; and a_affirm concludes s_red, a premise of nothing.)
	if !slices.Contains(chainEdges, Edge{EdgeChains, "a_play", "a_addition"}) {
		t.Errorf("missing chains edge a_play → a_addition; got %v", chainEdges)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	u := ballUniverse()
	u.Layout = map[string]Point{"s_red": {X: 120, Y: 80}}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Universe
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	v1, err := Evaluate(u)
	if err != nil {
		t.Fatalf("Evaluate original: %v", err)
	}
	v2, err := Evaluate(&back)
	if err != nil {
		t.Fatalf("Evaluate round-tripped: %v", err)
	}
	j1, _ := json.Marshal(v1)
	j2, _ := json.Marshal(v2)
	if string(j1) != string(j2) {
		t.Errorf("verdicts changed across JSON round-trip:\n%s\n%s", j1, j2)
	}
}

func TestValidateRejectsBrokenUniverses(t *testing.T) {
	cases := map[string]*Universe{
		"unknown ref": {
			Statements: []Statement{{ID: "s1", Text: "x"}},
			Formulas:   []Formula{{ID: "f1", Op: OpNot, Args: []string{"nope"}}},
		},
		"bad arity": {
			Statements: []Statement{{ID: "s1", Text: "x"}, {ID: "s2", Text: "y"}},
			Formulas:   []Formula{{ID: "f1", Op: OpNot, Args: []string{"s1", "s2"}}},
		},
		"duplicate id": {
			Statements: []Statement{{ID: "s1", Text: "x"}, {ID: "s1", Text: "y"}},
		},
		"formula cycle": {
			Statements: []Statement{{ID: "s1", Text: "x"}},
			Formulas: []Formula{
				{ID: "f1", Op: OpAnd, Args: []string{"s1", "f2"}},
				{ID: "f2", Op: OpNot, Args: []string{"f1"}},
			},
		},
		"argument without premises": {
			Statements: []Statement{{ID: "s1", Text: "x"}},
			Arguments:  []Argument{{ID: "a1", Conclusion: "s1"}},
		},
		"unknown op": {
			Statements: []Statement{{ID: "s1", Text: "x"}, {ID: "s2", Text: "y"}},
			Formulas:   []Formula{{ID: "f1", Op: "MAYBE", Args: []string{"s1", "s2"}}},
		},
	}
	for name, u := range cases {
		if err := Validate(u); err == nil {
			t.Errorf("%s: Validate should have failed", name)
		}
	}
	if err := Validate(ballUniverse()); err != nil {
		t.Errorf("ball universe should validate: %v", err)
	}
}
