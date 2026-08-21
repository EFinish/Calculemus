package core

import "testing"

// Every Boolean-valid syllogism mood must validate AND get its traditional
// name — built programmatically from figure and mood so the table in
// forms.go is exercised entry by entry, in both premise orders.

func stmtLetter(id string, letter byte, subj, pred string) Statement {
	switch letter {
	case 'A':
		return stmt(id, subj, QuantAll, pred, QualIs)
	case 'E':
		return stmt(id, subj, QuantNone, pred, QualIs)
	case 'I':
		return stmt(id, subj, QuantSome, pred, QualIs)
	default: // O
		return stmt(id, subj, QuantSome, pred, QualIsNot)
	}
}

func TestAllBooleanValidMoods(t *testing.T) {
	const S, P, M = "greeks", "mortal", "men"
	// Term positions per figure: {major subj, major pred, minor subj, minor pred}.
	figures := map[byte][4]string{
		'1': {M, P, S, M},
		'2': {P, M, S, M},
		'3': {M, P, M, S},
		'4': {P, M, M, S},
	}
	for key, want := range syllogismNames {
		fig, maj, min, con := key[0], key[1], key[2], key[3]
		pos := figures[fig]
		major := stmtLetter("major", maj, pos[0], pos[1])
		minor := stmtLetter("minor", min, pos[2], pos[3])
		concl := stmtLetter("concl", con, S, P)
		for _, prem := range [][]string{{"major", "minor"}, {"minor", "major"}} {
			u := &Universe{
				Version:    1,
				Statements: []Statement{major, minor, concl},
				Arguments:  []Argument{{ID: "a", Title: want, Premises: prem, Conclusion: "concl"}},
			}
			v, err := Evaluate(u)
			if err != nil {
				t.Fatalf("%s: %v", want, err)
			}
			if !v.Arguments[0].Valid {
				t.Errorf("%s (%s): must be valid under the Boolean reading", want, key)
				continue
			}
			if v.Arguments[0].Form != want {
				t.Errorf("%s (%s, premises %v): Form = %q", want, key, prem, v.Arguments[0].Form)
			}
		}
	}
}

// The import-needing moods stay invalid (Boolean reading) and therefore
// unlabeled — classifyForm only ever sees valid arguments.
func TestImportMoodsStayInvalid(t *testing.T) {
	cases := map[string][3]Statement{
		// Felapton (EAO-3): no men are gods, all men are greeks ⊢ some greeks are not gods.
		"Felapton": {
			stmt("major", "men", QuantNone, "gods", QualIs),
			stmt("minor", "men", QuantAll, "greeks", QualIs),
			stmt("concl", "greeks", QuantSome, "gods", QualIsNot),
		},
		// Bamalip (AAI-4): all gods are men, all men are mortal ⊢ some mortal are gods.
		"Bamalip": {
			stmt("major", "gods", QuantAll, "men", QualIs),
			stmt("minor", "men", QuantAll, "mortal", QualIs),
			stmt("concl", "mortal", QuantSome, "gods", QualIs),
		},
	}
	for name, ss := range cases {
		u := &Universe{
			Version:    1,
			Statements: []Statement{ss[0], ss[1], ss[2]},
			Arguments:  []Argument{{ID: "a", Title: name, Premises: []string{"major", "minor"}, Conclusion: "concl"}},
		}
		v, err := Evaluate(u)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if v.Arguments[0].Valid {
			t.Errorf("%s needs existential import and must be invalid here", name)
		}
	}
}
