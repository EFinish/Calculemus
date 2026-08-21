package core

// Tseitin compilation: every ref (statement or formula) gets a solver
// variable, and each formula contributes clauses forcing its variable to
// equal the truth of its connective over its argument variables. Queries are
// then the definitional clauses plus assumption unit literals — asserting a
// formula is one positive unit on its variable, negating a conclusion is one
// negative unit.

// A literal is ±var (1-based); a clause is a disjunction of literals.
type clause []int

type compiled struct {
	ix    *index
	varOf map[string]int
	nVars int
	defs  []clause
	terms *termSystem
	// regionVars[component][mask] is the "this Venn region is inhabited"
	// variable; index 0 (the all-absent region) is unused.
	regionVars [][]int
	// grounded is non-nil in relational (M6) mode; its domain size is
	// surfaced as Verdicts.BoundedDomain.
	grounded *grounding
	kindVars map[string][]int   // kind -> var per domain element
	verbVars map[string][][]int // verb -> var per (subject, object) element pair
}

func (c *compiled) fresh() int {
	c.nVars++
	return c.nVars
}

// andLit / orLit: a literal equivalent to the conjunction / disjunction of
// lits, via a fresh defined variable (single literals pass through).
func (c *compiled) andLit(lits []int) int {
	if len(lits) == 1 {
		return lits[0]
	}
	v := c.fresh()
	c.defineAnd(v, lits)
	return v
}

func (c *compiled) orLit(lits []int) int {
	if len(lits) == 1 {
		return lits[0]
	}
	v := c.fresh()
	c.defineOr(v, lits)
	return v
}

// encodeGrounded implements DESIGN §11: kinds as unary predicates and verbs
// as binary ones over a bounded domain, with every groundable statement's
// variable defined by its quantifier structure. Mirrors evalGroundedStmt —
// the property tests hold the two to account.
func (c *compiled) encodeGrounded() error {
	g, err := buildGrounding(c.ix.u)
	if err != nil {
		return err
	}
	c.grounded = g
	c.kindVars = make(map[string][]int, len(g.kinds))
	for _, k := range g.kinds {
		vars := make([]int, g.domain)
		for d := range vars {
			vars[d] = c.fresh()
		}
		c.kindVars[k] = vars
	}
	c.verbVars = make(map[string][][]int, len(g.verbs))
	for _, v := range g.verbs {
		grid := make([][]int, g.domain)
		for d := range grid {
			grid[d] = make([]int, g.domain)
			for e := range grid[d] {
				grid[d][e] = c.fresh()
			}
		}
		c.verbVars[v] = grid
	}

	for _, s := range g.stmts {
		lit := c.groundStatement(g, s)
		c.defineAnd(c.varOf[s.ID], []int{lit}) // statement var ↔ its grounding
	}
	return nil
}

// groundStatement returns a literal equivalent to the statement's truth over
// the bounded domain.
func (c *compiled) groundStatement(g *grounding, s *Statement) int {
	pred := func(d int) int {
		if s.Verb == "" {
			return c.kindVars[termKey(s.Predicate)][d]
		}
		verb := c.verbVars[termKey(s.Verb)]
		if s.ObjectIsIndividual {
			return verb[d][g.indIndex[termKey(s.Predicate)]]
		}
		objKind := c.kindVars[termKey(s.Predicate)]
		lits := make([]int, 0, g.domain)
		switch s.ObjectQuantifier {
		case QuantAll: // ∀e obj(e) → V(d,e)
			for e := 0; e < g.domain; e++ {
				lits = append(lits, c.orLit([]int{-objKind[e], verb[d][e]}))
			}
			return c.andLit(lits)
		case QuantNone: // ∀e obj(e) → ¬V(d,e)
			for e := 0; e < g.domain; e++ {
				lits = append(lits, c.orLit([]int{-objKind[e], -verb[d][e]}))
			}
			return c.andLit(lits)
		default: // SOME: ∃e obj(e) ∧ V(d,e)
			for e := 0; e < g.domain; e++ {
				lits = append(lits, c.andLit([]int{objKind[e], verb[d][e]}))
			}
			return c.orLit(lits)
		}
	}

	universal, positive := posOf(s)
	sign := func(lit int) int {
		if positive {
			return lit
		}
		return -lit
	}
	if s.SubjectIsIndividual {
		return sign(pred(g.indIndex[termKey(s.Subject)]))
	}
	subjKind := c.kindVars[termKey(s.Subject)]
	lits := make([]int, 0, g.domain)
	if universal { // ∀d subj(d) → ±pred(d)
		for d := 0; d < g.domain; d++ {
			lits = append(lits, c.orLit([]int{-subjKind[d], sign(pred(d))}))
		}
		return c.andLit(lits)
	}
	for d := 0; d < g.domain; d++ { // ∃d subj(d) ∧ ±pred(d)
		lits = append(lits, c.andLit([]int{subjKind[d], sign(pred(d))}))
	}
	return c.orLit(lits)
}

func compile(ix *index) (*compiled, error) {
	c := &compiled{ix: ix, varOf: make(map[string]int)}
	// Statements first, in universe order, for deterministic var numbering.
	for i := range ix.u.Statements {
		c.nVars++
		c.varOf[ix.u.Statements[i].ID] = c.nVars
	}
	for i := range ix.u.Formulas {
		c.nVars++
		c.varOf[ix.u.Formulas[i].ID] = c.nVars
	}
	for i := range ix.u.Formulas {
		c.encode(&ix.u.Formulas[i])
	}

	// M6: any verb or individual switches the whole term layer to
	// bounded-domain grounding (relational.go). Otherwise the M4 Venn-region
	// encoding applies — exact, and unchanged for every pre-M6 universe.
	if relationalMode(ix.u) {
		if err := c.encodeGrounded(); err != nil {
			return nil, err
		}
		return c, nil
	}

	// M4: statements with term structure stop being free atoms — their
	// variables are defined over the Venn-region variables of their
	// component (see terms.go).
	ts, err := buildTermSystem(ix.u)
	if err != nil {
		return nil, err
	}
	c.terms = ts
	c.regionVars = make([][]int, len(ts.components))
	for ci := range ts.components {
		k := len(ts.components[ci].terms)
		vars := make([]int, 1<<k)
		for mask := 1; mask < 1<<k; mask++ {
			c.nVars++
			vars[mask] = c.nVars
		}
		c.regionVars[ci] = vars
	}
	for i := range ts.stmts {
		st := &ts.stmts[i]
		k := len(ts.components[st.component].terms)
		var lits []int
		for _, mask := range st.regionMasks(k) {
			m := c.regionVars[st.component][mask]
			if st.isUniversal() {
				lits = append(lits, -m) // A/E: every such region empty
			} else {
				lits = append(lits, m) // I/O: some such region inhabited
			}
		}
		if st.isUniversal() {
			c.defineAnd(c.varOf[st.stmtID], lits)
		} else {
			c.defineOr(c.varOf[st.stmtID], lits)
		}
	}
	return c, nil
}

func (c *compiled) lit(ref string) int { return c.varOf[ref] }

func (c *compiled) encode(f *Formula) {
	v := c.varOf[f.ID]
	args := make([]int, len(f.Args))
	for i, a := range f.Args {
		args[i] = c.varOf[a]
	}
	switch f.Op {
	case OpNot:
		c.defineAnd(v, []int{-args[0]})
	case OpAnd:
		c.defineAnd(v, args)
	case OpOr:
		c.defineOr(v, args)
	case OpNand: // v ↔ ¬(a1∧…) ≡ v ↔ (¬a1∨…)
		c.defineOr(v, negate(args))
	case OpNor: // v ↔ ¬(a1∨…) ≡ v ↔ (¬a1∧…)
		c.defineAnd(v, negate(args))
	case OpImplies: // v ↔ (¬a ∨ b)
		c.defineOr(v, []int{-args[0], args[1]})
	case OpIff, OpXnor:
		a, b := args[0], args[1]
		c.defs = append(c.defs,
			clause{-v, -a, b}, clause{-v, a, -b}, // v → a↔b
			clause{v, -a, -b}, clause{v, a, b}, // a↔b → v
		)
	case OpXor:
		a, b := args[0], args[1]
		c.defs = append(c.defs,
			clause{-v, a, b}, clause{-v, -a, -b}, // v → a⊕b
			clause{v, -a, b}, clause{v, a, -b}, // a⊕b → v
		)
	}
}

// defineAnd emits v ↔ (l1 ∧ … ∧ lk).
func (c *compiled) defineAnd(v int, lits []int) {
	long := clause{v}
	for _, l := range lits {
		c.defs = append(c.defs, clause{-v, l})
		long = append(long, -l)
	}
	c.defs = append(c.defs, long)
}

// defineOr emits v ↔ (l1 ∨ … ∨ lk).
func (c *compiled) defineOr(v int, lits []int) {
	long := clause{-v}
	for _, l := range lits {
		c.defs = append(c.defs, clause{v, -l})
		long = append(long, l)
	}
	c.defs = append(c.defs, long)
}

func negate(lits []int) []int {
	out := make([]int, len(lits))
	for i, l := range lits {
		out[i] = -l
	}
	return out
}

// sat reports whether the definitional clauses plus the given assumption
// unit literals are satisfiable; on success it returns a total assignment
// indexed by variable (assignment[v] is true iff v is true).
func (c *compiled) sat(assumptions ...int) (bool, []bool) {
	clauses := make([]clause, 0, len(c.defs)+len(assumptions))
	clauses = append(clauses, c.defs...)
	for _, a := range assumptions {
		clauses = append(clauses, clause{a})
	}
	return solve(clauses, c.nVars)
}
