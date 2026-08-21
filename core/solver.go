package core

// DPLL with occurrence-list unit propagation over an undo trail. When a
// literal becomes false, only the clauses that contain it are re-examined —
// the difference between this and rescanning every clause per round is what
// keeps the discoveries sweep (hundreds of entailment probes per evaluate)
// interactive. Still no watched literals or learning; the compiled/sat
// boundary keeps a mature solver a drop-in swap if a universe ever outgrows
// this (DESIGN.md §4).

// solve returns a satisfying assignment for the clauses over vars 1..nVars,
// or (false, nil). The returned slice is indexed by variable; index 0 unused.
func solve(clauses []clause, nVars int) (bool, []bool) {
	s := &solver{
		clauses: clauses,
		occ:     make([][]int32, 2*(nVars+1)),
		assign:  make([]int8, nVars+1),
		trail:   make([]int, 0, nVars),
	}
	for ci, cl := range clauses {
		if len(cl) == 0 {
			return false, nil
		}
		for _, l := range cl {
			s.occ[litIndex(l)] = append(s.occ[litIndex(l)], int32(ci))
		}
	}
	// Seed: initial unit clauses, then propagate to fixpoint.
	for _, cl := range clauses {
		if len(cl) == 1 && !s.set(cl[0]) {
			return false, nil
		}
	}
	if !s.propagate(0) || !s.search() {
		return false, nil
	}
	model := make([]bool, nVars+1)
	for v := 1; v <= nVars; v++ {
		model[v] = s.assign[v] == 1
	}
	return true, model
}

type solver struct {
	clauses []clause
	occ     [][]int32 // literal index -> clauses containing that literal
	assign  []int8    // 0 unassigned, 1 true, -1 false
	trail   []int     // assigned literals, in order
}

func litIndex(lit int) int {
	if lit > 0 {
		return 2 * lit
	}
	return 2*(-lit) + 1
}

func litVal(assign []int8, lit int) int8 {
	if lit > 0 {
		return assign[lit]
	}
	return -assign[-lit]
}

// set records lit as true; false means it was already forced the other way.
func (s *solver) set(lit int) bool {
	switch litVal(s.assign, lit) {
	case 1:
		return true
	case -1:
		return false
	}
	if lit > 0 {
		s.assign[lit] = 1
	} else {
		s.assign[-lit] = -1
	}
	s.trail = append(s.trail, lit)
	return true
}

func (s *solver) undo(mark int) {
	for i := mark; i < len(s.trail); i++ {
		l := s.trail[i]
		if l > 0 {
			s.assign[l] = 0
		} else {
			s.assign[-l] = 0
		}
	}
	s.trail = s.trail[:mark]
}

// propagate runs unit propagation for every literal on the trail from
// position start onward, visiting only clauses weakened by each assignment.
func (s *solver) propagate(start int) bool {
	for p := start; p < len(s.trail); p++ {
		falsified := -s.trail[p]
		for _, ci := range s.occ[litIndex(falsified)] {
			cl := s.clauses[ci]
			satisfied := false
			unassigned := 0
			var unit int
			for _, l := range cl {
				switch litVal(s.assign, l) {
				case 1:
					satisfied = true
				case 0:
					unassigned++
					unit = l
				}
				if satisfied {
					break
				}
			}
			if satisfied {
				continue
			}
			if unassigned == 0 {
				return false // clause fully falsified
			}
			if unassigned == 1 && !s.set(unit) {
				return false
			}
		}
	}
	return true
}

func (s *solver) search() bool {
	branch := 0
	for v := 1; v < len(s.assign); v++ {
		if s.assign[v] == 0 {
			branch = v
			break
		}
	}
	if branch == 0 {
		return true // total assignment, no conflicts
	}
	for _, lit := range []int{branch, -branch} {
		mark := len(s.trail)
		if s.set(lit) && s.propagate(mark) && s.search() {
			return true
		}
		s.undo(mark)
	}
	return false
}
