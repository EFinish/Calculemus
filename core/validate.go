package core

import (
	"fmt"
	"strings"
)

// index resolves the shared id namespace of statements and formulas and is
// the precondition for every query: buildIndex succeeds only on a
// structurally valid universe.
type index struct {
	u    *Universe
	stmt map[string]*Statement
	form map[string]*Formula
}

func buildIndex(u *Universe) (*index, error) {
	ix := &index{
		u:    u,
		stmt: make(map[string]*Statement, len(u.Statements)),
		form: make(map[string]*Formula, len(u.Formulas)),
	}
	var problems []string
	report := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	for i := range u.Statements {
		s := &u.Statements[i]
		if s.ID == "" {
			report("statement %d has empty id", i)
			continue
		}
		if ix.stmt[s.ID] != nil {
			report("duplicate statement id %q", s.ID)
			continue
		}
		ix.stmt[s.ID] = s
	}
	for i := range u.Formulas {
		f := &u.Formulas[i]
		if f.ID == "" {
			report("formula %d has empty id", i)
			continue
		}
		if ix.stmt[f.ID] != nil || ix.form[f.ID] != nil {
			report("duplicate id %q", f.ID)
			continue
		}
		ix.form[f.ID] = f
	}

	for _, f := range ix.form {
		if err := checkArity(f); err != nil {
			report("formula %q: %v", f.ID, err)
		}
		for _, arg := range f.Args {
			if !ix.isRef(arg) {
				report("formula %q references unknown id %q", f.ID, arg)
			}
		}
	}
	if len(problems) == 0 {
		if cycle := ix.findCycle(); cycle != "" {
			report("formula cycle involving %q (formulas must form a DAG)", cycle)
		}
	}

	for i, a := range u.Assertions {
		if !ix.isRef(a.Formula) {
			report("assertion %d references unknown id %q", i, a.Formula)
		}
	}
	for _, arg := range u.Arguments {
		if arg.ID == "" {
			report("argument %q has empty id", arg.Title)
		}
		if len(arg.Premises) == 0 {
			report("argument %q has no premises", arg.ID)
		}
		for _, p := range arg.Premises {
			if !ix.isRef(p) {
				report("argument %q premise references unknown id %q", arg.ID, p)
			}
		}
		if !ix.isRef(arg.Conclusion) {
			report("argument %q conclusion references unknown id %q", arg.ID, arg.Conclusion)
		}
	}
	seenScenario := map[string]bool{}
	for _, sc := range u.Scenarios {
		if seenScenario[sc.Name] {
			report("duplicate scenario name %q", sc.Name)
		}
		seenScenario[sc.Name] = true
		for ref := range sc.Toggles {
			if !ix.isRef(ref) {
				report("scenario %q toggles unknown id %q", sc.Name, ref)
			}
		}
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid universe:\n  %s", strings.Join(problems, "\n  "))
	}
	return ix, nil
}

// Validate checks a universe's structural integrity: unique ids, resolvable
// references, connective arities, and acyclic formulas.
func Validate(u *Universe) error {
	_, err := buildIndex(u)
	return err
}

func checkArity(f *Formula) error {
	n := len(f.Args)
	switch f.Op {
	case OpNot:
		if n != 1 {
			return fmt.Errorf("NOT takes exactly 1 argument, got %d", n)
		}
	case OpImplies, OpIff, OpXor, OpXnor:
		if n != 2 {
			return fmt.Errorf("%s takes exactly 2 arguments, got %d", f.Op, n)
		}
	case OpAnd, OpOr, OpNand, OpNor:
		if n < 2 {
			return fmt.Errorf("%s takes at least 2 arguments, got %d", f.Op, n)
		}
	default:
		return fmt.Errorf("unknown op %q", f.Op)
	}
	return nil
}

func (ix *index) isRef(id string) bool {
	return ix.stmt[id] != nil || ix.form[id] != nil
}

// findCycle returns the id of some formula on a reference cycle, or "".
func (ix *index) findCycle() string {
	const (
		visiting = 1
		done     = 2
	)
	state := make(map[string]int, len(ix.form))
	var visit func(id string) string
	visit = func(id string) string {
		f := ix.form[id]
		if f == nil { // statements are leaves
			return ""
		}
		switch state[id] {
		case visiting:
			return id
		case done:
			return ""
		}
		state[id] = visiting
		for _, arg := range f.Args {
			if hit := visit(arg); hit != "" {
				return hit
			}
		}
		state[id] = done
		return ""
	}
	for id := range ix.form {
		if hit := visit(id); hit != "" {
			return hit
		}
	}
	return ""
}

// atoms returns the set of statement ids a ref transitively mentions.
func (ix *index) atoms(ref string) map[string]bool {
	out := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		if ix.stmt[id] != nil {
			out[id] = true
			return
		}
		for _, arg := range ix.form[id].Args {
			walk(arg)
		}
	}
	walk(ref)
	return out
}

// eval computes a ref's truth under a total assignment of atoms. This is the
// definitional semantics; the solver must always agree with it (the oracle
// tests enforce that).
func (ix *index) eval(ref string, assign map[string]bool) bool {
	if ix.stmt[ref] != nil {
		return assign[ref]
	}
	f := ix.form[ref]
	vals := make([]bool, len(f.Args))
	for i, arg := range f.Args {
		vals[i] = ix.eval(arg, assign)
	}
	switch f.Op {
	case OpNot:
		return !vals[0]
	case OpAnd:
		return allTrue(vals)
	case OpOr:
		return anyTrue(vals)
	case OpImplies:
		return !vals[0] || vals[1]
	case OpIff, OpXnor:
		return vals[0] == vals[1]
	case OpXor:
		return vals[0] != vals[1]
	case OpNand:
		return !allTrue(vals)
	case OpNor:
		return !anyTrue(vals)
	}
	panic("unreachable: validated op " + string(f.Op))
}

func allTrue(vals []bool) bool {
	for _, v := range vals {
		if !v {
			return false
		}
	}
	return true
}

func anyTrue(vals []bool) bool {
	for _, v := range vals {
		if v {
			return true
		}
	}
	return false
}
