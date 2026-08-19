package core

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

// M4 dogfood (DESIGN §8): Barbara — all men are mortal… — validates from
// quantifier structure alone. No conditionals anywhere in the universe.

func stmt(id, subject string, q Quantifier, predicate string, ql Qualifier) Statement {
	return Statement{
		ID: id, Text: fmt.Sprintf("%s %s %s %s", q, subject, ql, predicate),
		Subject: subject, Quantifier: q, Predicate: predicate, Qualifier: ql,
	}
}

func TestBarbara(t *testing.T) {
	u := &Universe{
		Version: 1,
		Title:   "Barbara",
		Statements: []Statement{
			stmt("major", "men", QuantAll, "mortal", QualIs),
			stmt("minor", "greeks", QuantAll, "men", QualIs),
			stmt("concl", "greeks", QuantAll, "mortal", QualIs),
			stmt("converse", "mortal", QuantAll, "greeks", QualIs),
		},
		Arguments: []Argument{
			{ID: "barbara", Title: "Barbara", Premises: []string{"major", "minor"}, Conclusion: "concl"},
			{ID: "bad", Title: "Converse", Premises: []string{"major", "minor"}, Conclusion: "converse"},
		},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	byID := map[string]ArgumentVerdict{}
	for _, av := range v.Arguments {
		byID[av.ID] = av
	}
	if !byID["barbara"].Valid {
		t.Fatal("Barbara must validate from quantifier structure alone")
	}
	if byID["barbara"].Form != "Barbara" {
		t.Errorf("Form = %q, want Barbara", byID["barbara"].Form)
	}
	if byID["bad"].Valid {
		t.Fatal("the converse conclusion must be invalid")
	}
}

func TestClassicalSyllogisms(t *testing.T) {
	cases := []struct {
		name       string
		premises   []Statement
		conclusion Statement
		valid      bool
	}{
		{
			// Celarent (EAE-1): no mortal is a god, all men are mortal ⊢ no man is a god.
			name: "Celarent",
			premises: []Statement{
				stmt("p1", "mortal", QuantNone, "god", QualIs),
				stmt("p2", "men", QuantAll, "mortal", QualIs),
			},
			conclusion: stmt("c", "men", QuantNone, "god", QualIs),
			valid:      true,
		},
		{
			// Darii (AII-1): all men are mortal, some greeks are men ⊢ some greeks are mortal.
			name: "Darii",
			premises: []Statement{
				stmt("p1", "men", QuantAll, "mortal", QualIs),
				stmt("p2", "greeks", QuantSome, "men", QualIs),
			},
			conclusion: stmt("c", "greeks", QuantSome, "mortal", QualIs),
			valid:      true,
		},
		{
			// Ferio (EIO-1): no men are gods, some greeks are men ⊢ some greeks are not gods.
			name: "Ferio",
			premises: []Statement{
				stmt("p1", "men", QuantNone, "god", QualIs),
				stmt("p2", "greeks", QuantSome, "men", QualIs),
			},
			conclusion: stmt("c", "greeks", QuantSome, "god", QualIsNot),
			valid:      true,
		},
		{
			// Darapti (AAI-3) needs existential import — INVALID under the
			// Boolean reading: if there are no men, the premises hold and
			// the particular conclusion fails.
			name: "Darapti",
			premises: []Statement{
				stmt("p1", "men", QuantAll, "mortal", QualIs),
				stmt("p2", "men", QuantAll, "greek", QualIs),
			},
			conclusion: stmt("c", "greek", QuantSome, "mortal", QualIs),
			valid:      false,
		},
		{
			// NONE/IS_NOT double negation: "none of the men are not mortal"
			// is A-form; Barbara shape must still validate through it.
			name: "double negation A-form",
			premises: []Statement{
				stmt("p1", "men", QuantNone, "mortal", QualIsNot),
				stmt("p2", "greeks", QuantAll, "men", QualIs),
			},
			conclusion: stmt("c", "greeks", QuantAll, "mortal", QualIs),
			valid:      true,
		},
	}
	for _, tc := range cases {
		u := &Universe{Version: 1, Statements: append(slices.Clone(tc.premises), tc.conclusion)}
		u.Arguments = []Argument{{
			ID: "a", Title: tc.name,
			Premises:   []string{tc.premises[0].ID, tc.premises[1].ID},
			Conclusion: tc.conclusion.ID,
		}}
		v, err := Evaluate(u)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if v.Arguments[0].Valid != tc.valid {
			t.Errorf("%s: Valid = %v, want %v", tc.name, v.Arguments[0].Valid, tc.valid)
		}
	}
}

func TestTermContradictionDiagnosed(t *testing.T) {
	// "All of the ball is red" and "some of the ball is not red" cannot
	// coexist — a contradiction the engine could not see before M4, because
	// the two statements were independent atoms.
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			stmt("s1", "the ball", QuantAll, "red", QualIs),
			stmt("s2", "the ball", QuantSome, "red", QualIsNot),
		},
		Assertions: []Assertion{
			{Formula: "s1", Active: true},
			{Formula: "s2", Active: true},
		},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if v.Consistent {
		t.Fatal("A-form and O-form over the same terms must contradict")
	}
	if len(v.UnsatCore) != 2 {
		t.Errorf("UnsatCore = %v, want both statements", v.UnsatCore)
	}
}

func TestFormAnnotations(t *testing.T) {
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			{ID: "x", Text: "x"}, {ID: "y", Text: "y"}, {ID: "z", Text: "z"},
		},
		Formulas: []Formula{
			{ID: "xy", Op: OpImplies, Args: []string{"x", "y"}},
			{ID: "yz", Op: OpImplies, Args: []string{"y", "z"}},
			{ID: "xz", Op: OpImplies, Args: []string{"x", "z"}},
			{ID: "nx", Op: OpNot, Args: []string{"x"}},
			{ID: "ny", Op: OpNot, Args: []string{"y"}},
			{ID: "xvy", Op: OpOr, Args: []string{"x", "y"}},
		},
		Arguments: []Argument{
			{ID: "mp", Title: "MP", Premises: []string{"xy", "x"}, Conclusion: "y"},
			{ID: "mt", Title: "MT", Premises: []string{"xy", "ny"}, Conclusion: "nx"},
			{ID: "hs", Title: "HS", Premises: []string{"xy", "yz"}, Conclusion: "xz"},
			{ID: "ds", Title: "DS", Premises: []string{"xvy", "nx"}, Conclusion: "y"},
			// Valid but nameless: conjunction-free identity.
			{ID: "id", Title: "ID", Premises: []string{"x", "y"}, Conclusion: "x"},
			// Invalid MP shape must NOT be labeled.
			{ID: "bad", Title: "AC", Premises: []string{"xy", "y"}, Conclusion: "x"},
		},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"mp": "modus ponens", "mt": "modus tollens",
		"hs": "hypothetical syllogism", "ds": "disjunctive syllogism",
		"id": "", "bad": "",
	}
	for _, av := range v.Arguments {
		if av.Form != want[av.ID] {
			t.Errorf("%s: Form = %q, want %q", av.ID, av.Form, want[av.ID])
		}
	}
	for _, av := range v.Arguments {
		if av.ID == "bad" && av.Valid {
			t.Error("affirming the consequent must stay invalid")
		}
	}
}

func TestComponentSizeCap(t *testing.T) {
	u := &Universe{Version: 1}
	for i := 0; i < 13; i++ {
		u.Statements = append(u.Statements,
			stmt(fmt.Sprintf("s%d", i), fmt.Sprintf("t%d", i), QuantAll, fmt.Sprintf("t%d", i+1), QualIs))
	}
	if _, err := Evaluate(u); err == nil {
		t.Fatal("a 14-term linked component must be rejected, not attempted")
	}
}

// randomTermUniverse: structured statements over a small shared vocabulary,
// mixed with opaque atoms and formulas — the property-test counterpart of
// randomUniverse, checked against the region-enumerating oracle.
func randomTermUniverse(rng *rand.Rand) *Universe {
	u := &Universe{Version: 1, Title: "random terms"}
	terms := []string{"alpha", "beta", "gamma"}
	quants := []Quantifier{QuantAll, QuantSome, QuantNone}
	quals := []Qualifier{QualIs, QualIsNot}
	refs := []string{}

	nStructured := 2 + rng.Intn(4) // 2..5
	for i := 0; i < nStructured; i++ {
		id := fmt.Sprintf("t%d", i)
		u.Statements = append(u.Statements, stmt(id,
			terms[rng.Intn(len(terms))], quants[rng.Intn(len(quants))],
			terms[rng.Intn(len(terms))], quals[rng.Intn(len(quals))]))
		refs = append(refs, id)
	}
	for i := 0; i < rng.Intn(3); i++ { // 0..2 opaque atoms
		id := fmt.Sprintf("o%d", i)
		u.Statements = append(u.Statements, Statement{ID: id, Text: id})
		refs = append(refs, id)
	}
	ops := []Op{OpNot, OpAnd, OpOr, OpImplies, OpXor}
	for i := 0; i < rng.Intn(4); i++ { // 0..3 formulas
		id := fmt.Sprintf("f%d", i)
		op := ops[rng.Intn(len(ops))]
		arity := 2
		if op == OpNot {
			arity = 1
		}
		args := make([]string, arity)
		for j := range args {
			args[j] = refs[rng.Intn(len(refs))]
		}
		u.Formulas = append(u.Formulas, Formula{ID: id, Op: op, Args: args})
		refs = append(refs, id)
	}
	for i, n := 0, rng.Intn(4); i < n; i++ {
		u.Assertions = append(u.Assertions, Assertion{Formula: refs[rng.Intn(len(refs))], Active: true})
	}
	for i, n := 0, rng.Intn(3); i < n; i++ {
		nPrem := 1 + rng.Intn(2)
		premises := make([]string, nPrem)
		for j := range premises {
			premises[j] = refs[rng.Intn(len(refs))]
		}
		u.Arguments = append(u.Arguments, Argument{
			ID: fmt.Sprintf("a%d", i), Premises: premises, Conclusion: refs[rng.Intn(len(refs))],
		})
	}
	return u
}

func TestTermSolverAgreesWithOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(1686)) // the year Leibniz's Discourse on Metaphysics circulated
	for iter := 0; iter < 200; iter++ {
		u := randomTermUniverse(rng)
		o := newOracle(t, u)
		active := activeAssertions(u, nil)

		v, err := Evaluate(u)
		if err != nil {
			t.Fatalf("iter %d: Evaluate: %v\nuniverse: %+v", iter, err, u)
		}
		if got, want := v.Consistent, o.satisfiable(active); got != want {
			t.Fatalf("iter %d: Consistent = %v, oracle says %v\nuniverse: %+v", iter, got, want, u)
		}
		if v.Consistent {
			for _, s := range u.Statements {
				wantTrue := o.entailed(active, s.ID, true)
				wantFalse := o.entailed(active, s.ID, false)
				if got := slices.Contains(v.EntailedTrue, s.ID); got != wantTrue {
					t.Fatalf("iter %d: EntailedTrue(%s) = %v, oracle says %v\nuniverse: %+v", iter, s.ID, got, wantTrue, u)
				}
				if got := slices.Contains(v.EntailedFalse, s.ID); got != wantFalse {
					t.Fatalf("iter %d: EntailedFalse(%s) = %v, oracle says %v\nuniverse: %+v", iter, s.ID, got, wantFalse, u)
				}
			}
		}
		for _, av := range v.Arguments {
			var arg *Argument
			for i := range u.Arguments {
				if u.Arguments[i].ID == av.ID {
					arg = &u.Arguments[i]
				}
			}
			if got, want := av.Valid, o.valid(arg); got != want {
				t.Fatalf("iter %d: argument %s Valid = %v, oracle says %v\nuniverse: %+v", iter, av.ID, got, want, u)
			}
		}
	}
}
