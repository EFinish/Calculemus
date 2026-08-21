package core

import (
	"fmt"
	"strings"
)

// M6: relations — the Frege step (DESIGN §11).
//
// When any statement uses a verb or an individual, the whole term layer
// switches from the Venn-region encoding (exact, monadic-only) to
// bounded-domain grounding: a domain of the named individuals plus a few
// anonymous witnesses, kinds as unary predicates over it, verbs as binary
// ones. Everything compiles to the same SAT engine.
//
// The declared honesty (guardrail R7): a countermodel found is a real
// countermodel; "valid" means "no countermodel with at most N things", and
// Verdicts.BoundedDomain reports N so the UI can say so. Named individuals
// are assumed distinct (unique-name assumption).

// The domain never exceeds this: kinds cost |D| variables each, verbs |D|².
const maxDomainSize = 8

const defaultWitnesses = 4

// relationalTrigger reports whether a statement uses the M6 grammar.
func relationalTrigger(s *Statement) bool {
	return s.Verb != "" || s.SubjectIsIndividual || s.ObjectIsIndividual
}

func relationalMode(u *Universe) bool {
	for i := range u.Statements {
		if relationalTrigger(&u.Statements[i]) {
			return true
		}
	}
	return false
}

// groundable: participates in grounded semantics (either full v1 term
// structure or the relational grammar). Anything else stays an opaque atom.
func groundable(s *Statement) bool {
	return structured(s) || relationalTrigger(s)
}

// grounding is the bounded domain and its symbol tables.
type grounding struct {
	domain      int            // |D| = len(individuals) + witnesses
	individuals []string       // normalized, first-seen order; element i is domain index i
	indIndex    map[string]int //
	kinds       []string       // normalized, first-seen order
	verbs       []string
	stmts       []*Statement // groundable statements, universe order
}

func buildGrounding(u *Universe) (*grounding, error) {
	g := &grounding{indIndex: map[string]int{}}
	seenKind := map[string]bool{}
	seenVerb := map[string]bool{}
	addInd := func(name string) {
		key := termKey(name)
		if _, ok := g.indIndex[key]; !ok {
			g.indIndex[key] = len(g.individuals)
			g.individuals = append(g.individuals, key)
		}
	}
	addKind := func(name string) {
		key := termKey(name)
		if !seenKind[key] {
			seenKind[key] = true
			g.kinds = append(g.kinds, key)
		}
	}
	for i := range u.Statements {
		s := &u.Statements[i]
		if !groundable(s) {
			continue
		}
		g.stmts = append(g.stmts, s)
		if s.SubjectIsIndividual {
			addInd(s.Subject)
		} else {
			addKind(s.Subject)
		}
		if s.ObjectIsIndividual {
			addInd(s.Predicate)
		} else {
			addKind(s.Predicate)
		}
		if s.Verb != "" {
			key := termKey(s.Verb)
			if !seenVerb[key] {
				seenVerb[key] = true
				g.verbs = append(g.verbs, key)
			}
		}
	}
	if len(g.individuals) > maxDomainSize {
		return nil, fmt.Errorf("%d named individuals exceed the domain cap of %d",
			len(g.individuals), maxDomainSize)
	}
	witnesses := u.Witnesses
	if witnesses <= 0 {
		witnesses = defaultWitnesses
	}
	if len(g.individuals)+witnesses > maxDomainSize {
		witnesses = maxDomainSize - len(g.individuals)
	}
	g.domain = len(g.individuals) + witnesses
	if g.domain == 0 {
		g.domain = 1 // a logic of nothing at all still needs one possible thing
	}
	return g, nil
}

// posOf: whether the statement affirms its predicate of the subject(s), and
// whether the subject quantification is universal. Individual subjects have
// no quantifier — only the qualifier's polarity matters.
func posOf(s *Statement) (universal, positive bool) {
	if s.SubjectIsIndividual {
		return false, s.Qualifier == QualIs
	}
	switch formOf(s) {
	case formA:
		return true, true
	case formE:
		return true, false
	case formI:
		return false, true
	default: // formO
		return false, false
	}
}

// --- oracle-side definitional semantics --------------------------------------
//
// evalGroundedStmt computes a statement's truth over an explicit world:
// kindVal(kind, d) and verbVal(verb, d, e) over domain indices 0..domain-1.
// Deliberately written as direct quantifier loops, independent of the CNF
// encoding, so the property tests check one against the other.
func evalGroundedStmt(g *grounding, s *Statement, kindVal func(kind string, d int) bool, verbVal func(verb string, d, e int) bool) bool {
	pred := func(d int) bool {
		if s.Verb == "" {
			return kindVal(termKey(s.Predicate), d)
		}
		verb := termKey(s.Verb)
		if s.ObjectIsIndividual {
			return verbVal(verb, d, g.indIndex[termKey(s.Predicate)])
		}
		objKind := termKey(s.Predicate)
		switch s.ObjectQuantifier {
		case QuantAll:
			for e := 0; e < g.domain; e++ {
				if kindVal(objKind, e) && !verbVal(verb, d, e) {
					return false
				}
			}
			return true
		case QuantNone:
			for e := 0; e < g.domain; e++ {
				if kindVal(objKind, e) && verbVal(verb, d, e) {
					return false
				}
			}
			return true
		default: // SOME
			for e := 0; e < g.domain; e++ {
				if kindVal(objKind, e) && verbVal(verb, d, e) {
					return true
				}
			}
			return false
		}
	}

	universal, positive := posOf(s)
	if s.SubjectIsIndividual {
		return pred(g.indIndex[termKey(s.Subject)]) == positive
	}
	subjKind := termKey(s.Subject)
	if universal {
		for d := 0; d < g.domain; d++ {
			if kindVal(subjKind, d) && pred(d) != positive {
				return false
			}
		}
		return true
	}
	for d := 0; d < g.domain; d++ {
		if kindVal(subjKind, d) && pred(d) == positive {
			return true
		}
	}
	return false
}

// --- validation ---------------------------------------------------------------

func validateRelationalStatement(s *Statement) error {
	if strings.TrimSpace(s.Subject) == "" || strings.TrimSpace(s.Predicate) == "" {
		return fmt.Errorf("relational statements need both a subject and an object")
	}
	if s.Qualifier != QualIs && s.Qualifier != QualIsNot {
		return fmt.Errorf("qualifier must be IS or IS_NOT")
	}
	if !s.SubjectIsIndividual {
		if q := s.Quantifier; q != QuantAll && q != QuantSome && q != QuantNone {
			return fmt.Errorf("kind subjects need a quantifier (ALL/SOME/NONE)")
		}
	}
	if s.Verb == "" {
		if s.ObjectIsIndividual {
			return fmt.Errorf("a copula object must be a kind, not an individual (identity is out of scope)")
		}
		if s.ObjectQuantifier != "" {
			return fmt.Errorf("copula statements quantify the subject only")
		}
		return nil
	}
	if !s.ObjectIsIndividual {
		if q := s.ObjectQuantifier; q != QuantAll && q != QuantSome && q != QuantNone {
			return fmt.Errorf("kind objects of a verb need an object quantifier (ALL/SOME/NONE)")
		}
	} else if s.ObjectQuantifier != "" {
		return fmt.Errorf("individual objects take no quantifier")
	}
	return nil
}
