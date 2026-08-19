package core

// Named-form annotation — the demotion promised in DESIGN §4: worldview-
// mapper's 983-line pattern-matching engine, reborn as a few dozen lines of
// purely decorative labeling. Matching is syntactic and strict; verdicts
// never depend on it (guardrail R1). An argument that matches no pattern is
// simply unlabeled.

// classifyForm names a valid argument's shape, or returns "".
func classifyForm(ix *index, arg *Argument) string {
	if len(arg.Premises) != 2 {
		if name := classifySyllogism(ix, arg); name != "" {
			return name
		}
		return ""
	}
	p0, p1 := ix.form[arg.Premises[0]], ix.form[arg.Premises[1]]
	concl := ix.form[arg.Conclusion]

	// Try both premise orders for the conditional patterns.
	for _, pair := range [][2]*Formula{{p0, p1}, {p1, p0}} {
		imp, other := pair[0], pair[1]
		otherID := arg.Premises[1]
		if imp == p1 {
			otherID = arg.Premises[0]
		}
		if imp == nil || imp.Op != OpImplies {
			continue
		}
		x, y := imp.Args[0], imp.Args[1]
		// Modus ponens: X→Y, X ⊢ Y
		if otherID == x && arg.Conclusion == y {
			return "modus ponens"
		}
		// Modus tollens: X→Y, ¬Y ⊢ ¬X
		if isNotOf(other, y) && isNotRef(ix, arg.Conclusion, x) {
			return "modus tollens"
		}
		// Hypothetical syllogism: X→Y, Y→Z ⊢ X→Z
		if other != nil && other.Op == OpImplies && concl != nil && concl.Op == OpImplies {
			if imp.Args[1] == other.Args[0] && concl.Args[0] == imp.Args[0] && concl.Args[1] == other.Args[1] {
				return "hypothetical syllogism"
			}
		}
	}
	// Disjunctive syllogism: X∨Y, ¬X ⊢ Y (either disjunct, either order).
	for _, pair := range [][2]int{{0, 1}, {1, 0}} {
		or := ix.form[arg.Premises[pair[0]]]
		neg := ix.form[arg.Premises[pair[1]]]
		if or == nil || or.Op != OpOr || len(or.Args) != 2 || neg == nil || neg.Op != OpNot {
			continue
		}
		if neg.Args[0] == or.Args[0] && arg.Conclusion == or.Args[1] {
			return "disjunctive syllogism"
		}
		if neg.Args[0] == or.Args[1] && arg.Conclusion == or.Args[0] {
			return "disjunctive syllogism"
		}
	}
	return classifySyllogism(ix, arg)
}

func isNotOf(f *Formula, ref string) bool {
	return f != nil && f.Op == OpNot && f.Args[0] == ref
}

func isNotRef(ix *index, id string, ref string) bool {
	return isNotOf(ix.form[id], ref)
}

// classifySyllogism recognizes Barbara (AAA-1), the syllogism of syllogisms:
// all M are P, all S are M ⊢ all S are P — from term structure alone.
func classifySyllogism(ix *index, arg *Argument) string {
	if len(arg.Premises) != 2 {
		return ""
	}
	a := aForm(ix, arg.Premises[0])
	b := aForm(ix, arg.Premises[1])
	c := aForm(ix, arg.Conclusion)
	if a == nil || b == nil || c == nil {
		return ""
	}
	for _, pair := range [][2]*[2]string{{a, b}, {b, a}} {
		major, minor := pair[0], pair[1]
		// major: M→P, minor: S→M, conclusion: S→P
		if minor[1] == major[0] && c[0] == minor[0] && c[1] == major[1] {
			return "Barbara"
		}
	}
	return ""
}

// aForm returns [subjectTerm, predicateTerm] when the ref is an A-form
// structured statement, else nil.
func aForm(ix *index, id string) *[2]string {
	s := ix.stmt[id]
	if s == nil || !structured(s) || formOf(s) != formA {
		return nil
	}
	return &[2]string{termKey(s.Subject), termKey(s.Predicate)}
}
