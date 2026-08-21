package core

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

// M6 dogfood (DESIGN §11): the boy/ball/red example that exposed term
// logic's wall. With relations, the connection needs no hand-written
// conditional — it is derived.

func indStmt(id, subject, verb, object string, oq Quantifier, ql Qualifier) Statement {
	s := Statement{
		ID: id, Text: fmt.Sprintf("%s %s %s", subject, verb, object),
		Subject: subject, SubjectIsIndividual: true,
		Verb: verb, Predicate: object, Qualifier: ql,
	}
	if oq != "" {
		s.ObjectQuantifier = oq
	} else {
		s.ObjectIsIndividual = true
	}
	return s
}

func fregeUniverse() *Universe {
	return &Universe{
		Version: 2,
		Title:   "The Frege step",
		Statements: []Statement{
			// the boy throws the ball
			indStmt("s_throws", "the boy", "throws", "the ball", "", QualIs),
			// the ball is red (copula, individual subject)
			{ID: "s_red", Text: "the ball is red", Subject: "the ball",
				SubjectIsIndividual: true, Qualifier: QualIs, Predicate: "red"},
			// the boy throws some red (the old conditional's conclusion, now first-class)
			indStmt("s_some_red", "the boy", "throws", "red", QuantSome, QualIs),
			// the boy throws none red
			indStmt("s_none_red", "the boy", "throws", "red", QuantNone, QualIs),
		},
		Arguments: []Argument{
			{ID: "a_frege", Title: "The Frege step",
				Premises: []string{"s_throws", "s_red"}, Conclusion: "s_some_red"},
			{ID: "a_gap", Title: "Without the color premise",
				Premises: []string{"s_throws"}, Conclusion: "s_some_red"},
		},
	}
}

func TestFregeStep(t *testing.T) {
	u := fregeUniverse()
	u.Assertions = []Assertion{
		{Formula: "s_throws", Active: true},
		{Formula: "s_red", Active: true},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !v.Consistent {
		t.Fatal("universe should be consistent")
	}
	// THE payoff: throws(boy, ball) ∧ red(ball) forces "throws something red"
	// with no conditional anywhere in the universe.
	if !slices.Contains(v.EntailedTrue, "s_some_red") {
		t.Errorf("s_some_red should be entailed; EntailedTrue = %v", v.EntailedTrue)
	}
	if !slices.Contains(v.EntailedFalse, "s_none_red") {
		t.Errorf("s_none_red should be entailed false; EntailedFalse = %v", v.EntailedFalse)
	}
	byID := map[string]ArgumentVerdict{}
	for _, av := range v.Arguments {
		byID[av.ID] = av
	}
	if !byID["a_frege"].Valid {
		t.Error("throws(boy,ball), red(ball) ⊢ throws-something-red must be valid")
	}
	// Without the color premise the inference must fail — the ball needn't
	// be red, and the countermodel says so.
	if byID["a_gap"].Valid {
		t.Error("throws(boy,ball) alone must not entail throwing something red")
	}
	// 2 individuals + 4 default witnesses.
	if v.BoundedDomain != 6 {
		t.Errorf("BoundedDomain = %d, want 6", v.BoundedDomain)
	}
}

func TestFregeContradiction(t *testing.T) {
	u := fregeUniverse()
	u.Assertions = []Assertion{
		{Formula: "s_throws", Active: true},
		{Formula: "s_red", Active: true},
		{Formula: "s_none_red", Active: true},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if v.Consistent {
		t.Fatal("throws(boy,ball) ∧ red(ball) ∧ throws-none-red must contradict")
	}
	if len(v.UnsatCore) != 3 {
		t.Errorf("UnsatCore = %v, want all three assertions", v.UnsatCore)
	}
}

// Relational Barbara: all men throw some ball; socrates is a man ⊢ socrates
// throws some ball. Kind-quantified subjects and objects compose with
// individuals.
func TestQuantifiedRelational(t *testing.T) {
	u := &Universe{
		Version: 2,
		Statements: []Statement{
			{ID: "all_throw", Text: "all men throw some ball",
				Subject: "men", Quantifier: QuantAll, Qualifier: QualIs,
				Verb: "throw", Predicate: "ball", ObjectQuantifier: QuantSome},
			{ID: "soc_man", Text: "socrates is a man",
				Subject: "socrates", SubjectIsIndividual: true, Qualifier: QualIs, Predicate: "men"},
			{ID: "soc_throws", Text: "socrates throws some ball",
				Subject: "socrates", SubjectIsIndividual: true, Qualifier: QualIs,
				Verb: "throw", Predicate: "ball", ObjectQuantifier: QuantSome},
		},
		Arguments: []Argument{
			{ID: "a", Title: "Relational instantiation",
				Premises: []string{"all_throw", "soc_man"}, Conclusion: "soc_throws"},
		},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Arguments[0].Valid {
		t.Fatal("universal instantiation through a verb must be valid")
	}
}

func TestNegatedVerb(t *testing.T) {
	// "the boy does not throw the ball" contradicts "the boy throws the ball".
	u := &Universe{
		Version: 2,
		Statements: []Statement{
			indStmt("pos", "the boy", "throws", "the ball", "", QualIs),
			indStmt("neg", "the boy", "throws", "the ball", "", QualIsNot),
		},
		Assertions: []Assertion{
			{Formula: "pos", Active: true},
			{Formula: "neg", Active: true},
		},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if v.Consistent {
		t.Fatal("throws ∧ does-not-throw must contradict")
	}
}

func TestWitnessesFieldHonored(t *testing.T) {
	u := fregeUniverse()
	u.Witnesses = 1
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if v.BoundedDomain != 3 { // 2 individuals + 1 witness
		t.Errorf("BoundedDomain = %d, want 3", v.BoundedDomain)
	}
}

func TestDomainCap(t *testing.T) {
	u := &Universe{Version: 2}
	for i := 0; i < 9; i++ {
		u.Statements = append(u.Statements, Statement{
			ID: fmt.Sprintf("s%d", i), Text: "x",
			Subject: fmt.Sprintf("thing %d", i), SubjectIsIndividual: true,
			Qualifier: QualIs, Predicate: "red",
		})
	}
	if _, err := Evaluate(u); err == nil {
		t.Fatal("9 named individuals must exceed the domain cap")
	}
}

func TestRelationalValidation(t *testing.T) {
	bad := map[string]Statement{
		"copula with individual object": {ID: "s", Text: "x", Subject: "a",
			SubjectIsIndividual: true, Qualifier: QualIs,
			Predicate: "b", ObjectIsIndividual: true},
		"verb kind object without quantifier": {ID: "s", Text: "x", Subject: "a",
			SubjectIsIndividual: true, Qualifier: QualIs, Verb: "sees", Predicate: "cats"},
		"kind subject without quantifier": {ID: "s", Text: "x", Subject: "men",
			Qualifier: QualIs, Verb: "sees", Predicate: "cats", ObjectQuantifier: QuantSome},
		"individual object with quantifier": {ID: "s", Text: "x", Subject: "a",
			SubjectIsIndividual: true, Qualifier: QualIs, Verb: "sees",
			Predicate: "b", ObjectIsIndividual: true, ObjectQuantifier: QuantSome},
	}
	for name, s := range bad {
		u := &Universe{Version: 2, Statements: []Statement{s}}
		if err := Validate(u); err == nil {
			t.Errorf("%s: Validate should have failed", name)
		}
	}
}

// randomRelationalUniverse: tiny vocabulary so the grounded oracle's bit
// budget stays enumerable: at most 1 individual, 2 kinds, 1 verb, 1 witness.
func randomRelationalUniverse(rng *rand.Rand) *Universe {
	u := &Universe{Version: 2, Title: "random relational", Witnesses: 1}
	kinds := []string{"kappa", "lambda"}
	quants := []Quantifier{QuantAll, QuantSome, QuantNone}
	quals := []Qualifier{QualIs, QualIsNot}
	refs := []string{}

	n := 2 + rng.Intn(3) // 2..4 statements
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("r%d", i)
		s := Statement{ID: id, Text: id, Qualifier: quals[rng.Intn(2)]}
		if rng.Intn(2) == 0 {
			s.Subject, s.SubjectIsIndividual = "iota", true
		} else {
			s.Subject, s.Quantifier = kinds[rng.Intn(2)], quants[rng.Intn(3)]
		}
		if rng.Intn(2) == 0 { // copula
			s.Predicate = kinds[rng.Intn(2)]
		} else {
			s.Verb = "loves"
			if rng.Intn(2) == 0 {
				s.Predicate, s.ObjectIsIndividual = "iota", true
			} else {
				s.Predicate, s.ObjectQuantifier = kinds[rng.Intn(2)], quants[rng.Intn(3)]
			}
		}
		u.Statements = append(u.Statements, s)
		refs = append(refs, id)
	}
	if rng.Intn(2) == 0 {
		u.Statements = append(u.Statements, Statement{ID: "op", Text: "op"})
		refs = append(refs, "op")
	}
	ops := []Op{OpNot, OpAnd, OpOr, OpImplies}
	for i := 0; i < rng.Intn(3); i++ {
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
	for i, n := 0, rng.Intn(3); i < n; i++ {
		u.Assertions = append(u.Assertions, Assertion{Formula: refs[rng.Intn(len(refs))], Active: true})
	}
	for i, n := 0, rng.Intn(3); i < n; i++ {
		u.Arguments = append(u.Arguments, Argument{
			ID: fmt.Sprintf("a%d", i), Premises: []string{refs[rng.Intn(len(refs))]},
			Conclusion: refs[rng.Intn(len(refs))],
		})
	}
	return u
}

func TestGroundedSolverAgreesWithOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(1879)) // Begriffsschrift
	for iter := 0; iter < 200; iter++ {
		u := randomRelationalUniverse(rng)
		if !relationalMode(u) {
			continue // pure-copula draw; covered by the Venn property test
		}
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
