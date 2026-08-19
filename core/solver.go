package core

import "slices"

// A ~60-line DPLL: unit propagation plus chronological backtracking.
// Hand-built universes have dozens of atoms, so this is comfortably enough
// for years (DESIGN.md §4); the compiled/sat boundary makes a mature solver
// a drop-in swap if that ever changes.

// solve returns a satisfying assignment for the clauses over vars 1..nVars,
// or (false, nil). The returned slice is indexed by variable; index 0 unused.
func solve(clauses []clause, nVars int) (bool, []bool) {
	assign := make([]int8, nVars+1) // 0 unassigned, 1 true, -1 false
	if !dpll(clauses, assign) {
		return false, nil
	}
	model := make([]bool, nVars+1)
	for v := 1; v <= nVars; v++ {
		model[v] = assign[v] == 1
	}
	return true, model
}

func dpll(clauses []clause, assign []int8) bool {
	// Unit propagation to fixpoint.
	for {
		propagated := false
		for _, cl := range clauses {
			satisfied := false
			var unit int
			units := 0
			for _, lit := range cl {
				switch litVal(assign, lit) {
				case 1:
					satisfied = true
				case 0:
					unit = lit
					units++
				}
				if satisfied {
					break
				}
			}
			if satisfied {
				continue
			}
			if units == 0 {
				return false // conflict: clause fully falsified
			}
			if units == 1 {
				setLit(assign, unit)
				propagated = true
			}
		}
		if !propagated {
			break
		}
	}

	branch := 0
	for v := 1; v < len(assign); v++ {
		if assign[v] == 0 {
			branch = v
			break
		}
	}
	if branch == 0 {
		return true // total assignment, no conflicts
	}
	for _, sign := range []int8{1, -1} {
		trial := slices.Clone(assign)
		trial[branch] = sign
		if dpll(clauses, trial) {
			copy(assign, trial)
			return true
		}
	}
	return false
}

func litVal(assign []int8, lit int) int8 {
	if lit > 0 {
		return assign[lit]
	}
	return -assign[-lit]
}

func setLit(assign []int8, lit int) {
	if lit > 0 {
		assign[lit] = 1
	} else {
		assign[-lit] = -1
	}
}
