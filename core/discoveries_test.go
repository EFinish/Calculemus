package core

import (
	"fmt"
	"math/rand"
	"testing"
)

func discoveryKeys(ds []Statement) map[string]bool {
	keys := map[string]bool{}
	for i := range ds {
		s := &ds[i]
		keys[fmt.Sprintf("%s|%s|%s|%s|%s|%v|%v|%s",
			s.Subject, s.Quantifier, s.Qualifier, s.Verb, s.Predicate,
			s.SubjectIsIndividual, s.ObjectIsIndividual, s.ObjectQuantifier)] = true
	}
	return keys
}

func TestDiscoveryJeff(t *testing.T) {
	u := &Universe{
		Version: 2,
		Statements: []Statement{
			{ID: "s1", Text: "all lions are mammals",
				Subject: "lions", Quantifier: QuantAll, Qualifier: QualIs, Predicate: "mammals"},
			{ID: "s2", Text: "Jeff the Lion is a lion",
				Subject: "jeff the lion", SubjectIsIndividual: true, Qualifier: QualIs, Predicate: "lions"},
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
	keys := discoveryKeys(v.Discoveries)
	want := "jeff the lion||IS||mammals|true|false|"
	if !keys[want] {
		t.Fatalf("expected discovery 'jeff is mammals'; got %+v", v.Discoveries)
	}

	// Adopting the discovery removes it from future discoveries.
	u.Statements = append(u.Statements, Statement{
		ID: "s3", Text: "Jeff the Lion is a mammal",
		Subject: "jeff the lion", SubjectIsIndividual: true, Qualifier: QualIs, Predicate: "mammals",
	})
	v, err = Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if discoveryKeys(v.Discoveries)[want] {
		t.Fatal("adopted discovery must not be re-proposed")
	}
}

func TestDiscoveryNegativePolarity(t *testing.T) {
	// red(ball) + no red is blue ⇒ discover "the ball is not blue".
	u := &Universe{
		Version: 2,
		Statements: []Statement{
			{ID: "s1", Text: "the ball is red",
				Subject: "ball", SubjectIsIndividual: true, Qualifier: QualIs, Predicate: "red"},
			{ID: "s2", Text: "no red is blue",
				Subject: "red", Quantifier: QuantNone, Qualifier: QualIs, Predicate: "blue"},
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
	if !discoveryKeys(v.Discoveries)["ball||IS_NOT||blue|true|false|"] {
		t.Fatalf("expected discovery 'the ball is not blue'; got %+v", v.Discoveries)
	}
}

func TestDiscoverySuppressedWhenInconsistent(t *testing.T) {
	u := &Universe{
		Version: 2,
		Statements: []Statement{
			{ID: "s1", Text: "x", Subject: "ball", SubjectIsIndividual: true, Qualifier: QualIs, Predicate: "red"},
			{ID: "s2", Text: "y", Subject: "ball", SubjectIsIndividual: true, Qualifier: QualIsNot, Predicate: "red"},
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
	if v.Consistent || len(v.Discoveries) != 0 {
		t.Fatalf("no discoveries under a contradiction; got %+v", v.Discoveries)
	}
}

func TestDiscoveryVacuityFilter(t *testing.T) {
	// lions forced empty: "all lions are X" holds for every X — noise, not news.
	u := &Universe{
		Version: 1,
		Statements: []Statement{
			{ID: "s1", Text: "all lions are fierce",
				Subject: "lions", Quantifier: QuantAll, Qualifier: QualIs, Predicate: "fierce"},
			{ID: "s2", Text: "no lions are fierce",
				Subject: "lions", Quantifier: QuantNone, Qualifier: QualIs, Predicate: "fierce"},
			{ID: "s3", Text: "some cats are fierce",
				Subject: "cats", Quantifier: QuantSome, Qualifier: QualIs, Predicate: "fierce"},
		},
		Assertions: []Assertion{
			{Formula: "s1", Active: true},
			{Formula: "s2", Active: true},
			{Formula: "s3", Active: true},
		},
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Consistent {
		t.Fatal("empty lions are consistent (Boolean reading)")
	}
	for i := range v.Discoveries {
		d := &v.Discoveries[i]
		if termKey(d.Subject) == "lions" && (formOf(d) == formA || formOf(d) == formE) {
			t.Fatalf("vacuous universal about empty lions leaked: %+v", d)
		}
	}
}

// Every discovery must actually be entailed — checked against the oracle by
// appending it to the universe as a statement.
func TestDiscoveriesSoundVsOracle(t *testing.T) {
	rngs := map[string]func(*rand.Rand) *Universe{
		"venn":       randomTermUniverse,
		"relational": randomRelationalUniverse,
	}
	for name, gen := range rngs {
		rng := rand.New(rand.NewSource(1900)) // Russell's paradox found ~1901; close enough
		for iter := 0; iter < 120; iter++ {
			u := gen(rng)
			v, err := Evaluate(u)
			if err != nil {
				t.Fatalf("%s iter %d: %v", name, iter, err)
			}
			if !v.Consistent {
				continue
			}
			for di := range v.Discoveries {
				aug := *u
				d := v.Discoveries[di]
				d.ID, d.Text = "@check", "check"
				aug.Statements = append(append([]Statement{}, u.Statements...), d)
				o := newOracle(t, &aug)
				if !o.entailed(activeAssertions(u, nil), "@check", true) {
					t.Fatalf("%s iter %d: unsound discovery %+v\nuniverse: %+v", name, iter, d, u)
				}
			}
		}
	}
}
