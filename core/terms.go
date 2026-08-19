package core

import (
	"fmt"
	"strings"
)

// M4: syllogistic semantics for the ALL/SOME/NONE quantifiers apriorio's
// grammar has carried since M0 (guardrail R5).
//
// A statement with full term structure is no longer an opaque atom: its
// subject and predicate name monadic predicates over an implicit domain, and
// its truth is defined by which Venn regions of that domain are inhabited.
// Monadic logic has the small-model property, so "region r is nonempty" as a
// propositional variable per region is a sound and complete encoding — the
// quantifiers compile down to the same SAT engine as everything else.
//
// Reading: modern/Boolean — universal statements carry no existential import
// ("all unicorns are white" is true when there are no unicorns, matching the
// vacuous-truth stance of DESIGN §4.2). Barbara is valid either way; Darapti
// and friends, which need non-empty terms, are not.

// The four Aristotelian forms, derived from (quantifier, qualifier).
type syllogisticForm int

const (
	formA syllogisticForm = iota // all S are P
	formE                        // no S is P
	formI                        // some S are P
	formO                        // some S are not P
)

// structured reports whether a statement carries full term structure. Anything
// less stays an opaque propositional atom, exactly as before M4.
func structured(s *Statement) bool {
	quantOK := s.Quantifier == QuantAll || s.Quantifier == QuantSome || s.Quantifier == QuantNone
	qualOK := s.Qualifier == QualIs || s.Qualifier == QualIsNot
	return strings.TrimSpace(s.Subject) != "" && strings.TrimSpace(s.Predicate) != "" && quantOK && qualOK
}

func formOf(s *Statement) syllogisticForm {
	neg := s.Qualifier == QualIsNot
	switch s.Quantifier {
	case QuantAll:
		if neg {
			return formE // "all S is not P" = no S is P
		}
		return formA
	case QuantNone:
		if neg {
			return formA // "none of S is not P" = all S are P
		}
		return formE
	default: // SOME
		if neg {
			return formO
		}
		return formI
	}
}

// Terms are matched by normalized name; subjects and predicates share one
// namespace (the "mortal" in a predicate is the "mortal" in a subject).
func termKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// structuredStmt is one statement's place in the term system.
type structuredStmt struct {
	stmtID    string
	form      syllogisticForm
	component int
	subj      int // term index within the component
	pred      int
}

// termComponent is a connected component of terms (linked by co-occurring in
// a statement). Regions are subsets of the component's terms; only regions
// within a component interact, which keeps 2^k tractable.
type termComponent struct {
	terms []string // normalized names, deterministic first-seen order
}

type termSystem struct {
	components []termComponent
	stmts      []structuredStmt
}

// Components above this size would need 2^k region variables; a hand-built
// universe never gets close, and failing loudly beats hanging.
const maxComponentTerms = 12

// buildTermSystem partitions the structured statements' terms into connected
// components via union-find, deterministically (universe statement order).
func buildTermSystem(u *Universe) (*termSystem, error) {
	parent := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	ensure := func(x string) {
		if _, ok := parent[x]; !ok {
			parent[x] = x
		}
	}

	type sp struct{ id, subj, pred string }
	var pairs []sp
	var order []string // first-seen term order
	seen := map[string]bool{}
	for i := range u.Statements {
		s := &u.Statements[i]
		if !structured(s) {
			continue
		}
		subj, pred := termKey(s.Subject), termKey(s.Predicate)
		for _, t := range []string{subj, pred} {
			ensure(t)
			if !seen[t] {
				seen[t] = true
				order = append(order, t)
			}
		}
		parent[find(subj)] = find(pred)
		pairs = append(pairs, sp{s.ID, subj, pred})
	}
	if len(pairs) == 0 {
		return &termSystem{}, nil
	}

	ts := &termSystem{}
	compOf := map[string]int{}   // root term -> component index
	termIdx := map[string]int{}  // term -> index within its component
	termComp := map[string]int{} // term -> component index
	for _, t := range order {
		root := find(t)
		ci, ok := compOf[root]
		if !ok {
			ci = len(ts.components)
			compOf[root] = ci
			ts.components = append(ts.components, termComponent{})
		}
		c := &ts.components[ci]
		termIdx[t] = len(c.terms)
		termComp[t] = ci
		c.terms = append(c.terms, t)
	}
	for _, c := range ts.components {
		if len(c.terms) > maxComponentTerms {
			return nil, fmt.Errorf(
				"terms %v are all linked through shared statements: %d terms need 2^%d regions (max %d terms per linked group)",
				c.terms, len(c.terms), len(c.terms), maxComponentTerms)
		}
	}
	for i := range u.Statements {
		s := &u.Statements[i]
		if !structured(s) {
			continue
		}
		subj, pred := termKey(s.Subject), termKey(s.Predicate)
		ts.stmts = append(ts.stmts, structuredStmt{
			stmtID:    s.ID,
			form:      formOf(s),
			component: termComp[subj],
			subj:      termIdx[subj],
			pred:      termIdx[pred],
		})
	}
	return ts, nil
}

// regionsFor enumerates the (non-empty-subset) region masks relevant to a
// statement: those containing the subject term, split by predicate
// membership. The all-absent region can never matter — every form quantifies
// over elements that are S.
//
// A: true iff every S∧¬P region is empty      → AND of ¬m over in(S), out(P)
// E: true iff every S∧P region is empty       → AND of ¬m over in(S), in(P)
// I: true iff some S∧P region is inhabited    → OR of m over in(S), in(P)
// O: true iff some S∧¬P region is inhabited   → OR of m over in(S), out(P)
func (st *structuredStmt) regionMasks(k int) []int {
	predIn := st.form == formE || st.form == formI
	var masks []int
	for mask := 1; mask < 1<<k; mask++ {
		if mask&(1<<st.subj) == 0 {
			continue
		}
		if (mask&(1<<st.pred) != 0) == predIn {
			masks = append(masks, mask)
		}
	}
	return masks
}

func (st *structuredStmt) isUniversal() bool {
	return st.form == formA || st.form == formE
}

// evalStructured computes a structured statement's truth from a set of
// inhabited region masks — the definitional semantics, used by the oracle.
func (st *structuredStmt) evalStructured(k int, inhabited func(mask int) bool) bool {
	for _, mask := range st.regionMasks(k) {
		if inhabited(mask) {
			return !st.isUniversal() // a witness satisfies I/O, refutes A/E
		}
	}
	return st.isUniversal() // no witness: A/E hold vacuously, I/O fail
}
