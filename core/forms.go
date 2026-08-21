package core

import "fmt"

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

// syllogismNames: figure+mood → traditional name, Boolean-valid moods only.
// The nine that need existential import (Darapti, Felapton, Bamalip, Fesapo,
// and the subalterns) are absent by design — under the Boolean reading those
// arguments are invalid, so classifyForm never sees them anyway.
var syllogismNames = map[string]string{
	"1AAA": "Barbara", "1EAE": "Celarent", "1AII": "Darii", "1EIO": "Ferio",
	"2EAE": "Cesare", "2AEE": "Camestres", "2EIO": "Festino", "2AOO": "Baroco",
	"3IAI": "Disamis", "3AII": "Datisi", "3OAO": "Bocardo", "3EIO": "Ferison",
	"4AEE": "Calemes", "4IAI": "Dimatis", "4EIO": "Fresison",
}

// classifySyllogism names a categorical syllogism by figure and mood: the
// conclusion fixes S and P, the shared middle term M and its position pick
// the figure, and the three form letters spell the mood. Matching stays
// positional ("some M are P" and "some P are M" classify differently even
// though they mean the same) — strictness over cleverness, per the header.
func classifySyllogism(ix *index, arg *Argument) string {
	if len(arg.Premises) != 2 {
		return ""
	}
	concl := catForm(ix, arg.Conclusion)
	if concl == nil {
		return ""
	}
	s, p := concl.subj, concl.pred
	if s == p {
		return ""
	}
	for _, order := range [][2]string{
		{arg.Premises[0], arg.Premises[1]},
		{arg.Premises[1], arg.Premises[0]},
	} {
		major, minor := catForm(ix, order[0]), catForm(ix, order[1])
		if major == nil || minor == nil {
			continue
		}
		figure := 0
		switch {
		case major.subj != s && major.subj != p && major.pred == p &&
			minor.subj == s && minor.pred == major.subj:
			figure = 1 // M-P, S-M
		case major.subj == p && major.pred != s && major.pred != p &&
			minor.subj == s && minor.pred == major.pred:
			figure = 2 // P-M, S-M
		case major.subj != s && major.subj != p && major.pred == p &&
			minor.subj == major.subj && minor.pred == s:
			figure = 3 // M-P, M-S
		case major.subj == p && major.pred != s && major.pred != p &&
			minor.subj == major.pred && minor.pred == s:
			figure = 4 // P-M, M-S
		default:
			continue
		}
		key := fmt.Sprintf("%d%c%c%c", figure, formLetter(major.form),
			formLetter(minor.form), formLetter(concl.form))
		if name := syllogismNames[key]; name != "" {
			return name
		}
	}
	return ""
}

type catStmt struct {
	form       syllogisticForm
	subj, pred string
}

// catForm returns a statement's categorical shape, or nil when the ref is
// not a structured copular statement. Relational statements are excluded:
// formOf ignores the verb, and "all men THROW balls" is not a categorical
// premise however it quantifies.
func catForm(ix *index, id string) *catStmt {
	s := ix.stmt[id]
	if s == nil || relationalTrigger(s) || !structured(s) {
		return nil
	}
	return &catStmt{formOf(s), termKey(s.Subject), termKey(s.Predicate)}
}

func formLetter(f syllogisticForm) byte {
	return [...]byte{formA: 'A', formE: 'E', formI: 'I', formO: 'O'}[f]
}
