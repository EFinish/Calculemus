package core

import "fmt"

// Discoveries — saturation (DESIGN spirit, refined): the machine never
// AUTHORS structure, but it may PROPOSE it. computeDiscoveries sweeps the
// atomic statements expressible in the universe's existing vocabulary
// (individuals × kinds, individuals × verbs × phrases, kind pairs in the
// four categorical forms), and reports the ones the active assertions force
// — in either polarity — that the user never wrote. The UI offers each with
// an adopt button; only adoption makes it part of the universe.
//
// Noise filters: candidates already covered by an authored statement are
// skipped, as are universal claims about kinds the assertions force empty
// (vacuously true of everything, informative about nothing).

const (
	maxDiscoveryCandidates = 1200
	maxDiscoveries         = 24
)

// computeDiscoveries assumes the universe is consistent (the caller guards —
// under a contradiction everything is "discovered", per §4.1).
func computeDiscoveries(u *Universe, activeRefs []string) []Statement {
	cands := candidateStatements(u)
	if len(cands) == 0 {
		return nil
	}
	if len(cands) > maxDiscoveryCandidates {
		cands = cands[:maxDiscoveryCandidates]
	}

	// One augmented compile: candidates are conservative definitions over the
	// existing vocabulary, so they never change what the assertions allow.
	aug := *u
	aug.Statements = append(append([]Statement{}, u.Statements...), cands...)
	ix, err := buildIndex(&aug)
	if err != nil {
		return nil // a malformed candidate is a bug, not a user error; stay silent
	}
	c, err := compile(ix)
	if err != nil {
		return nil // e.g. candidate would exceed a size cap — no discoveries then
	}
	assume := make([]int, len(activeRefs))
	for i, ref := range activeRefs {
		assume[i] = c.lit(ref)
	}
	ok, firstModel := c.sat(assume...)
	if !ok {
		return nil // caller guards, but never derive from a contradiction
	}
	e := newEntailer(c, assume, firstModel)

	// Kinds the assertions force empty: universal claims about them are
	// vacuous noise. Probed via the "@empty:" I(k,k) candidates.
	forcedEmpty := map[string]bool{}
	for i := range cands {
		s := &cands[i]
		if len(s.ID) > 7 && s.ID[:7] == "@empty:" {
			forcedEmpty[termKey(s.Subject)] = e.forcedFalse(c.lit(s.ID)) // ¬∃k forced
		}
	}

	var out []Statement
	report := func(s Statement) {
		s.ID, s.Text = "", "" // the app assigns both on adoption
		out = append(out, s)
	}
	for i := range cands {
		if len(out) >= maxDiscoveries {
			break
		}
		s := cands[i]
		if len(s.ID) > 7 && s.ID[:7] == "@empty:" {
			continue // probe, not a proposal
		}
		v := c.lit(s.ID)
		vacuous := !s.SubjectIsIndividual && forcedEmpty[termKey(s.Subject)]
		if e.forcedTrue(v) {
			// Universal forms over an empty subject are true of everything.
			if vacuous && (formOf(&s) == formA || formOf(&s) == formE) {
				continue
			}
			report(s)
			continue
		}
		if e.forcedFalse(v) {
			flipped, ok := flipCandidate(s)
			if !ok || (vacuous && (formOf(&flipped) == formA || formOf(&flipped) == formE)) {
				continue
			}
			report(flipped)
		}
	}
	return out
}

// candidateStatements enumerates the unauthored atomic sentences of the
// existing vocabulary, plus "@empty:" probes (SOME k IS k) per kind.
// Positive-polarity shapes only — flipCandidate covers the rest at query
// time, so each semantic pair costs one variable.
func candidateStatements(u *Universe) []Statement {
	existing := existingShapeKeys(u)
	var cands []Statement
	n := 0
	add := func(key string, s Statement) {
		if existing[key] {
			return
		}
		if s.ID == "" { // "@empty:" probes carry their own marker id
			s.ID = fmt.Sprintf("@d%d", n)
			n++
		}
		cands = append(cands, s)
	}

	if relationalMode(u) {
		g, err := buildGrounding(u)
		if err != nil {
			return nil
		}
		// individual IS kind — the "Jeff the Lion is mammals" class.
		for _, i := range g.individuals {
			for _, k := range g.kinds {
				add("IC|"+i+"|"+k, Statement{
					Subject: i, SubjectIsIndividual: true, Qualifier: QualIs, Predicate: k,
				})
			}
		}
		// individual VERB individual.
		for _, verb := range g.verbs {
			for _, i := range g.individuals {
				for _, j := range g.individuals {
					add("IVI|"+i+"|"+verb+"|"+j, Statement{
						Subject: i, SubjectIsIndividual: true, Qualifier: QualIs,
						Verb: verb, Predicate: j, ObjectIsIndividual: true,
					})
				}
				// individual VERB {SOME, ALL} kind (NONE and not-all arrive
				// via flipCandidate).
				for _, k := range g.kinds {
					for _, q := range []Quantifier{QuantSome, QuantAll} {
						add("IVK|"+i+"|"+verb+"|"+k+"|"+objClass(q, QualIs), Statement{
							Subject: i, SubjectIsIndividual: true, Qualifier: QualIs,
							Verb: verb, Predicate: k, ObjectQuantifier: q,
						})
					}
				}
			}
		}
		addKindPairs(g.kinds, add, true)
		return cands
	}

	// Pure Venn mode: kind pairs within each linked component only — terms in
	// different components never constrain each other, and cross-component
	// candidates could blow the region cap.
	ts, err := buildTermSystem(u)
	if err != nil {
		return nil
	}
	for _, comp := range ts.components {
		addKindPairs(comp.terms, add, false)
	}
	return cands
}

// addKindPairs emits A(s,p) and I(s,p) candidates (E and O arrive via
// flipCandidate), plus one "@empty:" probe per kind.
func addKindPairs(kinds []string, add func(string, Statement), grounded bool) {
	for _, s := range kinds {
		add("@empty:"+s, Statement{
			ID:      "@empty:" + s, // marker id: probe, never proposed
			Subject: s, Quantifier: QuantSome, Qualifier: QualIs, Predicate: s,
		})
		for _, p := range kinds {
			if s == p {
				continue
			}
			add("KA|"+s+"|"+p, Statement{
				Subject: s, Quantifier: QuantAll, Qualifier: QualIs, Predicate: p,
			})
			// I is symmetric; emit one orientation.
			if s < p {
				add("KI|"+s+"|"+p, Statement{
					Subject: s, Quantifier: QuantSome, Qualifier: QualIs, Predicate: p,
				})
			}
		}
	}
	_ = grounded
}

// flipCandidate returns the statement expressing the NEGATION of s, when the
// grammar has a clean form for it.
func flipCandidate(s Statement) (Statement, bool) {
	f := s
	if s.Verb != "" && s.ObjectQuantifier != "" {
		switch s.ObjectQuantifier {
		case QuantSome: // ¬(throws some k) = throws none of k
			f.ObjectQuantifier = QuantNone
			return f, true
		case QuantAll: // ¬(throws all k) = does not throw all of k
			f.Qualifier = QualIsNot
			return f, true
		}
		return f, false
	}
	if s.SubjectIsIndividual || s.Verb != "" {
		// ¬(jeff is k) = jeff is not k; ¬(i verbs j) = i does not verb j.
		f.Qualifier = QualIsNot
		return f, true
	}
	switch formOf(&s) {
	case formA: // ¬A = O
		f.Quantifier, f.Qualifier = QuantSome, QualIsNot
		return f, true
	case formI: // ¬I = E
		f.Quantifier, f.Qualifier = QuantNone, QualIs
		return f, true
	}
	return f, false
}

// existingShapeKeys canonicalizes the authored statements so candidates the
// user already covers — in either polarity — are skipped. Their truth is
// already visible as ⊨ badges on the statements themselves.
func existingShapeKeys(u *Universe) map[string]bool {
	keys := map[string]bool{}
	for i := range u.Statements {
		s := &u.Statements[i]
		if !groundable(s) {
			continue
		}
		subj, pred := termKey(s.Subject), termKey(s.Predicate)
		switch {
		case s.Verb != "" && s.ObjectIsIndividual:
			keys["IVI|"+subj+"|"+termKey(s.Verb)+"|"+pred] = true
		case s.Verb != "":
			// Both polarity classes of the pair share one candidate: SOME/NONE
			// are one pair, ALL/not-ALL the other.
			cls := objClass(s.ObjectQuantifier, s.Qualifier)
			pair := map[string]string{"SOME": "SOME", "NONE": "SOME", "ALL": "ALL", "NOTALL": "ALL"}[cls]
			keys["IVK|"+subj+"|"+termKey(s.Verb)+"|"+pred+"|"+pair] = true
		case s.SubjectIsIndividual:
			keys["IC|"+subj+"|"+pred] = true
		default:
			switch formOf(s) {
			case formA, formO: // A/O share the KA candidate
				keys["KA|"+subj+"|"+pred] = true
			case formI, formE: // I/E share the KI candidate (symmetric)
				a, b := subj, pred
				if a > b {
					a, b = b, a
				}
				keys["KI|"+a+"|"+b] = true
			}
		}
	}
	return keys
}

// objClass names the four distinct meanings of (object quantifier ×
// qualifier) under sentential negation.
func objClass(q Quantifier, ql Qualifier) string {
	neg := ql == QualIsNot
	switch q {
	case QuantSome:
		if neg {
			return "NONE" // does not verb some k = verbs none
		}
		return "SOME"
	case QuantNone:
		if neg {
			return "SOME"
		}
		return "NONE"
	default: // ALL
		if neg {
			return "NOTALL"
		}
		return "ALL"
	}
}
