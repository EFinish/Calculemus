package core

import (
	"fmt"
	"slices"
	"sort"
)

// Belief revision as a meta-query (DESIGN.md §12): "what must I give up to
// believe X?" answered with the same SAT primitive as everything else. The
// logic never retracts — each answer is a classical satisfiability fact
// about which assertion subsets can coexist with the target. This is AGM's
// remainder machinery: retraction sets are complements of the maximal
// assertion subsets consistent with the target, and the USER is the
// selection function — the machine proposes prices, only the user pays one.

const (
	maxRetractionSets  = 8
	maxRetractionDepth = 4
	maxReviseProbes    = 400
)

// Revision reports what holding a target true (or false) would cost.
type Revision struct {
	// Possible: the target belief has at least one model at all. When false,
	// no retraction can help — the belief contradicts itself.
	Possible bool `json:"possible"`
	// AlreadySatisfiable: the active assertions permit the belief as-is.
	AlreadySatisfiable bool `json:"alreadySatisfiable,omitempty"`
	// Retractions: minimal sets of active assertion refs — remove any one
	// whole set and the belief becomes consistent with what remains. Sorted
	// smallest-first; capped, so absence of a set is not proof there is none.
	Retractions [][]string `json:"retractions,omitempty"`
	// BoundedDomain mirrors Verdicts.BoundedDomain (M6 semantics caveat).
	BoundedDomain int `json:"boundedDomain,omitempty"`
}

// Revise computes the minimal retraction sets for holding target (a
// statement or formula id) true or false under the named scenario (""
// = the base assertions).
func Revise(u *Universe, scenario string, target string, wantTrue bool) (*Revision, error) {
	var toggles map[string]bool
	if scenario != "" {
		i := slices.IndexFunc(u.Scenarios, func(sc Scenario) bool { return sc.Name == scenario })
		if i < 0 {
			return nil, fmt.Errorf("unknown scenario %q", scenario)
		}
		toggles = u.Scenarios[i].Toggles
	}
	ix, err := buildIndex(u)
	if err != nil {
		return nil, err
	}
	if ix.stmt[target] == nil && ix.form[target] == nil {
		return nil, fmt.Errorf("unknown target %q", target)
	}
	c, err := compile(ix)
	if err != nil {
		return nil, err
	}
	lit := c.lit(target)
	if !wantTrue {
		lit = -lit
	}

	r := &Revision{}
	if c.grounded != nil {
		r.BoundedDomain = c.grounded.domain
	}
	if ok, _ := c.sat(lit); !ok {
		return r, nil // impossible in every world; Possible stays false
	}
	r.Possible = true

	active := activeAssertions(u, toggles)
	// Holding the target IS one of the assertions? Then the question is about
	// the others — a belief never has to retract itself.
	active = slices.DeleteFunc(slices.Clone(active), func(ref string) bool {
		return wantTrue && ref == target
	})
	if satWith(c, active, lit) {
		r.AlreadySatisfiable = true
		return r, nil
	}
	r.Retractions = enumerateRetractions(c, active, lit)
	return r, nil
}

func satWith(c *compiled, refs []string, hard int) bool {
	lits := make([]int, 0, len(refs)+1)
	lits = append(lits, hard)
	for _, ref := range refs {
		lits = append(lits, c.lit(ref))
	}
	ok, _ := c.sat(lits...)
	return ok
}

// enumerateRetractions walks the core-guided branching tree: whenever the
// kept assertions still contradict the target, some member of a minimal
// core must go — branch on each. Satisfiable leaves are retraction sets;
// subset-filtering keeps only the minimal ones.
func enumerateRetractions(c *compiled, active []string, lit int) [][]string {
	var found [][]string
	probes := 0
	var walk func(removed []string)
	walk = func(removed []string) {
		if len(found) >= maxRetractionSets || probes >= maxReviseProbes {
			return
		}
		// A superset of a known retraction set can't be minimal.
		for _, f := range found {
			if containsAll(removed, f) {
				return
			}
		}
		kept := slices.DeleteFunc(slices.Clone(active), func(ref string) bool {
			return slices.Contains(removed, ref)
		})
		probes++
		if satWith(c, kept, lit) {
			set := slices.Clone(removed)
			sort.Strings(set)
			if !slices.ContainsFunc(found, func(f []string) bool { return slices.Equal(f, set) }) {
				found = append(found, set)
			}
			return
		}
		if len(removed) >= maxRetractionDepth {
			return
		}
		for _, ref := range minimizeCore(c, kept, lit) {
			walk(append(slices.Clone(removed), ref))
		}
	}
	walk(nil)

	// Drop non-minimal stragglers (a small set found late invalidates big
	// ones found early), then order smallest-first, ties by content.
	var minimal [][]string
	for _, s := range found {
		keep := true
		for _, t := range found {
			if len(t) < len(s) && containsAll(s, t) {
				keep = false
				break
			}
		}
		if keep {
			minimal = append(minimal, s)
		}
	}
	sort.Slice(minimal, func(i, j int) bool {
		if len(minimal[i]) != len(minimal[j]) {
			return len(minimal[i]) < len(minimal[j])
		}
		return slices.Compare(minimal[i], minimal[j]) < 0
	})
	return minimal
}

func containsAll(super, sub []string) bool {
	for _, s := range sub {
		if !slices.Contains(super, s) {
			return false
		}
	}
	return true
}
