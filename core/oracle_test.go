package core

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

// The oracle: brute-force enumeration of every assignment over the atoms,
// evaluated with the definitional semantics in index.eval. For small
// universes it is trivially correct, so the solver pipeline
// (Tseitin + DPLL + queries) must agree with it on everything — guardrail
// R4's foundation.

type regionBit struct {
	component int
	mask      int
}

type oracle struct {
	ix    *index
	atoms []string // OPAQUE statement ids, universe order
	// Venn mode (pure M4): one enumeration bit per region per component.
	ts      *termSystem
	regions []regionBit
	// Grounded mode (M6): one bit per kind×element and per verb×pair.
	g *grounding
}

func newOracle(t *testing.T, u *Universe) *oracle {
	t.Helper()
	ix, err := buildIndex(u)
	if err != nil {
		t.Fatalf("buildIndex: %v", err)
	}
	o := &oracle{ix: ix}
	if relationalMode(u) {
		g, err := buildGrounding(u)
		if err != nil {
			t.Fatalf("buildGrounding: %v", err)
		}
		o.g = g
		for i := range u.Statements {
			if !groundable(&u.Statements[i]) {
				o.atoms = append(o.atoms, u.Statements[i].ID)
			}
		}
		bits := len(o.atoms) + len(g.kinds)*g.domain + len(g.verbs)*g.domain*g.domain
		if bits > 18 {
			t.Fatalf("grounded oracle limited to 18 bits, got %d", bits)
		}
		return o
	}
	ts, err := buildTermSystem(u)
	if err != nil {
		t.Fatalf("buildTermSystem: %v", err)
	}
	o.ts = ts
	for i := range u.Statements {
		if !structured(&u.Statements[i]) {
			o.atoms = append(o.atoms, u.Statements[i].ID)
		}
	}
	for ci := range ts.components {
		k := len(ts.components[ci].terms)
		for mask := 1; mask < 1<<k; mask++ {
			o.regions = append(o.regions, regionBit{ci, mask})
		}
	}
	if len(o.atoms)+len(o.regions) > 18 {
		t.Fatalf("oracle limited to 18 bits, got %d atoms + %d regions",
			len(o.atoms), len(o.regions))
	}
	return o
}

// forEachModel visits every world under which all refs hold. In Venn mode a
// world is opaque atoms × inhabited regions; in grounded mode it is opaque
// atoms × a full extension for every kind and verb over the bounded domain.
// Term-structured statements get their truth derived, never enumerated.
func (o *oracle) forEachModel(refs []string, visit func(assign map[string]bool)) {
	nA := len(o.atoms)
	derive := func(bits int, assign map[string]bool) {}
	total := nA

	if o.g != nil {
		g := o.g
		kindBase := nA
		verbBase := nA + len(g.kinds)*g.domain
		total = verbBase + len(g.verbs)*g.domain*g.domain
		kindIdx := map[string]int{}
		for i, k := range g.kinds {
			kindIdx[k] = i
		}
		verbIdx := map[string]int{}
		for i, v := range g.verbs {
			verbIdx[v] = i
		}
		derive = func(bits int, assign map[string]bool) {
			kindVal := func(kind string, d int) bool {
				return bits&(1<<(kindBase+kindIdx[kind]*g.domain+d)) != 0
			}
			verbVal := func(verb string, d, e int) bool {
				return bits&(1<<(verbBase+verbIdx[verb]*g.domain*g.domain+d*g.domain+e)) != 0
			}
			for _, s := range g.stmts {
				assign[s.ID] = evalGroundedStmt(g, s, kindVal, verbVal)
			}
		}
	} else {
		total = nA + len(o.regions)
		derive = func(bits int, assign map[string]bool) {
			inhabited := func(ci, mask int) bool {
				for j, r := range o.regions {
					if r.component == ci && r.mask == mask {
						return bits&(1<<(nA+j)) != 0
					}
				}
				return false
			}
			for i := range o.ts.stmts {
				st := &o.ts.stmts[i]
				k := len(o.ts.components[st.component].terms)
				assign[st.stmtID] = st.evalStructured(k, func(mask int) bool {
					return inhabited(st.component, mask)
				})
			}
		}
	}

	for bits := 0; bits < 1<<total; bits++ {
		assign := make(map[string]bool, len(o.ix.u.Statements))
		for i, id := range o.atoms {
			assign[id] = bits&(1<<i) != 0
		}
		derive(bits, assign)
		holds := true
		for _, ref := range refs {
			if !o.ix.eval(ref, assign) {
				holds = false
				break
			}
		}
		if holds {
			visit(assign)
		}
	}
}

func (o *oracle) satisfiable(refs []string) bool {
	found := false
	o.forEachModel(refs, func(map[string]bool) { found = true })
	return found
}

// entailed reports whether refs force target to the given truth value.
// Callers only ask this of satisfiable ref sets.
func (o *oracle) entailed(refs []string, target string, value bool) bool {
	forced := true
	o.forEachModel(refs, func(assign map[string]bool) {
		if o.ix.eval(target, assign) != value {
			forced = false
		}
	})
	return forced
}

func (o *oracle) valid(arg *Argument) bool {
	valid := true
	o.forEachModel(arg.Premises, func(assign map[string]bool) {
		if !o.ix.eval(arg.Conclusion, assign) {
			valid = false
		}
	})
	return valid
}

// randomUniverse builds a structurally valid universe: formula args only
// reference earlier ids, so the DAG constraint holds by construction.
func randomUniverse(rng *rand.Rand) *Universe {
	u := &Universe{Version: 1, Title: "random"}
	refs := []string{}
	nStatements := 3 + rng.Intn(4) // 3..6
	for i := 0; i < nStatements; i++ {
		id := fmt.Sprintf("s%d", i)
		u.Statements = append(u.Statements, Statement{ID: id, Text: id})
		refs = append(refs, id)
	}
	ops := []Op{OpNot, OpAnd, OpOr, OpImplies, OpIff, OpXor, OpNand, OpNor, OpXnor}
	nFormulas := rng.Intn(9) // 0..8
	for i := 0; i < nFormulas; i++ {
		id := fmt.Sprintf("f%d", i)
		op := ops[rng.Intn(len(ops))]
		var arity int
		switch op {
		case OpNot:
			arity = 1
		case OpImplies, OpIff, OpXor, OpXnor:
			arity = 2
		default:
			arity = 2 + rng.Intn(2) // 2..3
		}
		args := make([]string, arity)
		for j := range args {
			args[j] = refs[rng.Intn(len(refs))]
		}
		u.Formulas = append(u.Formulas, Formula{ID: id, Op: op, Args: args})
		refs = append(refs, id)
	}
	for i, n := 0, rng.Intn(5); i < n; i++ { // 0..4 assertions
		u.Assertions = append(u.Assertions, Assertion{
			Formula: refs[rng.Intn(len(refs))],
			Active:  rng.Intn(4) != 0, // mostly active
		})
	}
	for i, n := 0, rng.Intn(4); i < n; i++ { // 0..3 arguments
		nPrem := 1 + rng.Intn(3)
		premises := make([]string, nPrem)
		for j := range premises {
			premises[j] = refs[rng.Intn(len(refs))]
		}
		u.Arguments = append(u.Arguments, Argument{
			ID:         fmt.Sprintf("a%d", i),
			Premises:   premises,
			Conclusion: refs[rng.Intn(len(refs))],
		})
	}
	return u
}

func TestSolverAgreesWithOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(1646)) // Leibniz's birth year; fixed for reproducibility
	for iter := 0; iter < 500; iter++ {
		u := randomUniverse(rng)
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
			for _, f := range u.Formulas {
				if f.Op != OpImplies {
					continue
				}
				want := o.entailed(active, f.Args[0], false)
				if got := slices.Contains(v.Vacuous, f.ID); got != want {
					t.Fatalf("iter %d: Vacuous(%s) = %v, oracle says %v\nuniverse: %+v", iter, f.ID, got, want, u)
				}
			}
		} else {
			// The core must (a) come from the active set, (b) be
			// unsatisfiable, (c) be minimal: every one-smaller subset is
			// satisfiable.
			for _, ref := range v.UnsatCore {
				if !slices.Contains(active, ref) {
					t.Fatalf("iter %d: core member %q not among active assertions", iter, ref)
				}
			}
			if o.satisfiable(v.UnsatCore) {
				t.Fatalf("iter %d: reported core %v is satisfiable", iter, v.UnsatCore)
			}
			for i := range v.UnsatCore {
				sub := slices.Concat(v.UnsatCore[:i], v.UnsatCore[i+1:])
				if !o.satisfiable(sub) {
					t.Fatalf("iter %d: core %v not minimal — still unsat without %q", iter, v.UnsatCore, v.UnsatCore[i])
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
			if !av.Valid {
				// The countermodel must witness: premises hold, conclusion fails.
				for _, p := range arg.Premises {
					if !o.ix.eval(p, av.Countermodel) {
						t.Fatalf("iter %d: countermodel %v does not satisfy premise %q", iter, av.Countermodel, p)
					}
				}
				if o.ix.eval(arg.Conclusion, av.Countermodel) {
					t.Fatalf("iter %d: countermodel %v satisfies the conclusion %q", iter, av.Countermodel, arg.Conclusion)
				}
			}
		}
	}
}

func TestChainsEdgesAgreeWithOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(1716)) // and his death year
	for iter := 0; iter < 200; iter++ {
		u := randomUniverse(rng)
		if len(u.Arguments) < 2 {
			continue
		}
		o := newOracle(t, u)
		v, err := Evaluate(u)
		if err != nil {
			t.Fatalf("iter %d: Evaluate: %v", iter, err)
		}
		got := map[[2]string]bool{}
		for _, e := range v.Edges {
			if e.Type == EdgeChains {
				got[[2]string{e.From, e.To}] = true
			}
		}
		for i := range u.Arguments {
			for j := range u.Arguments {
				if i == j {
					continue
				}
				a, b := &u.Arguments[i], &u.Arguments[j]
				want := false
				for _, p := range b.Premises {
					if o.entailed([]string{a.Conclusion}, p, true) {
						want = true
						break
					}
				}
				if got[[2]string{a.ID, b.ID}] != want {
					t.Fatalf("iter %d: chains %s→%s = %v, oracle says %v\nuniverse: %+v",
						iter, a.ID, b.ID, got[[2]string{a.ID, b.ID}], want, u)
				}
			}
		}
	}
}

// entailed() with an unsatisfiable premise set is vacuously true for any
// target — the oracle-level view of ex falso quodlibet. Pin that down so the
// oracle itself isn't misread, and pin the engine's explosion guard against it.
func TestExplosionGuardVsOracle(t *testing.T) {
	u := &Universe{
		Version:    1,
		Statements: []Statement{{ID: "p", Text: "p"}, {ID: "q", Text: "q"}},
		Formulas:   []Formula{{ID: "np", Op: OpNot, Args: []string{"p"}}},
		Assertions: []Assertion{
			{Formula: "p", Active: true},
			{Formula: "np", Active: true},
		},
	}
	o := newOracle(t, u)
	if !o.entailed([]string{"p", "np"}, "q", true) {
		t.Fatal("oracle: a contradiction should vacuously entail anything")
	}
	v, err := Evaluate(u)
	if err != nil {
		t.Fatal(err)
	}
	if v.Consistent {
		t.Fatal("p ∧ ¬p should be inconsistent")
	}
	if len(v.EntailedTrue) != 0 {
		t.Fatalf("explosion guard failed: EntailedTrue = %v", v.EntailedTrue)
	}
	wantCore := []string{"np", "p"}
	gotCore := slices.Clone(v.UnsatCore)
	slices.Sort(gotCore)
	if !slices.Equal(gotCore, wantCore) {
		t.Fatalf("UnsatCore = %v, want %v", v.UnsatCore, wantCore)
	}
}
