package core

// entailer answers "is this literal forced by the assumptions?" with
// model-pool pruning: every satisfying assignment the solver returns is kept,
// and a candidate that any pooled model refutes is dismissed without a solver
// call. Each real SAT call therefore either proves an entailment (UNSAT) or
// contributes a new model that prunes later candidates — turning the
// entailment sweep and the discoveries grid from O(candidates) solver calls
// into roughly O(entailments found).
type entailer struct {
	c      *compiled
	assume []int
	pool   [][]bool
}

func newEntailer(c *compiled, assume []int, firstModel []bool) *entailer {
	e := &entailer{c: c, assume: assume}
	if firstModel != nil {
		e.pool = append(e.pool, firstModel)
	}
	return e
}

// forcedTrue: no model of the assumptions makes v false.
func (e *entailer) forcedTrue(v int) bool {
	for _, m := range e.pool {
		if !m[v] {
			return false
		}
	}
	ok, m := e.c.sat(append(append([]int{}, e.assume...), -v)...)
	if ok {
		e.pool = append(e.pool, m)
		return false
	}
	return true
}

// forcedFalse: no model of the assumptions makes v true.
func (e *entailer) forcedFalse(v int) bool {
	for _, m := range e.pool {
		if m[v] {
			return false
		}
	}
	ok, m := e.c.sat(append(append([]int{}, e.assume...), v)...)
	if ok {
		e.pool = append(e.pool, m)
		return false
	}
	return true
}
