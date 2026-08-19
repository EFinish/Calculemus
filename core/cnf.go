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
}

func compile(ix *index) *compiled {
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
	return c
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
