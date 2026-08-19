// Command calculemus evaluates a universe document and prints the verdicts —
// a terminal window onto the engine until the app exists (M1+).
//
// Usage:
//
//	calculemus <universe.json> [-scenario name] [-json]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/EFinish/Calculemus/core"
)

const (
	reset = "\033[0m"
	dim   = "\033[2m"
	green = "\033[32m"
	red   = "\033[31m"
	cyan  = "\033[36m"
)

func main() {
	scenario := flag.String("scenario", "", "evaluate under a named scenario's toggles")
	asJSON := flag.Bool("json", false, "print raw verdicts JSON instead of the report")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: calculemus <universe.json> [-scenario name] [-json]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fail(err)
	}
	var u core.Universe
	if err := json.Unmarshal(data, &u); err != nil {
		fail(fmt.Errorf("parsing %s: %w", flag.Arg(0), err))
	}

	var v *core.Verdicts
	if *scenario != "" {
		v, err = core.EvaluateScenario(&u, *scenario)
	} else {
		v, err = core.Evaluate(&u)
	}
	if err != nil {
		fail(err)
	}

	if *asJSON {
		out, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(out))
		return
	}
	report(&u, v, *scenario)
	if !v.Consistent {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "calculemus:", err)
	os.Exit(2)
}

func report(u *core.Universe, v *core.Verdicts, scenario string) {
	fmt.Printf("%s∎ %s%s", cyan, u.Title, reset)
	if scenario != "" {
		fmt.Printf("%s  (scenario: %s)%s", dim, scenario, reset)
	}
	fmt.Println()

	if v.Consistent {
		fmt.Printf("%s✓ universe is consistent%s\n", green, reset)
		if len(v.EntailedTrue)+len(v.EntailedFalse) > 0 {
			fmt.Println("\nDerived truths (forced by your assertions):")
			for _, id := range v.EntailedTrue {
				fmt.Printf("  %s⊨ TRUE %s %s\n", green, reset, render(u, id))
			}
			for _, id := range v.EntailedFalse {
				fmt.Printf("  %s⊨ FALSE%s %s\n", red, reset, render(u, id))
			}
		}
		for _, id := range v.Vacuous {
			fmt.Printf("%s  ⚠ %s holds only vacuously — its IF-part is forced false, so it says nothing here%s\n",
				dim, render(u, id), reset)
		}
	} else {
		fmt.Printf("%s⊥ universe is contradictory%s\n", red, reset)
		fmt.Println("\nMinimal conflict — these assertions cannot coexist; drop any one:")
		for _, ref := range v.UnsatCore {
			fmt.Printf("  %s✗%s %s\n", red, reset, render(u, ref))
		}
		fmt.Printf("%s  (derived truths suspended: a contradiction entails everything)%s\n", dim, reset)
	}

	if len(v.Arguments) > 0 {
		fmt.Println("\nArguments:")
		for _, av := range v.Arguments {
			arg := findArgument(u, av.ID)
			if av.Valid {
				fmt.Printf("  %sVALID  %s %s\n", green, reset, arg.Title)
			} else {
				fmt.Printf("  %sINVALID%s %s\n", red, reset, arg.Title)
				fmt.Printf("%s          countermodel — premises hold, conclusion fails, when:%s\n", dim, reset)
				for _, s := range u.Statements {
					fmt.Printf("%s            %-5v %s%s\n", dim, av.Countermodel[s.ID], s.Text, reset)
				}
			}
		}
	}

	var chains []core.Edge
	for _, e := range v.Edges {
		if e.Type == core.EdgeChains {
			chains = append(chains, e)
		}
	}
	if len(chains) > 0 {
		fmt.Println("\nChained arguments (conclusion feeds a premise — nobody drew these):")
		for _, e := range chains {
			fmt.Printf("  %s⊢→⊢%s %s → %s\n", cyan, reset, findArgument(u, e.From).Title, findArgument(u, e.To).Title)
		}
	}
}

func findArgument(u *core.Universe, id string) *core.Argument {
	for i := range u.Arguments {
		if u.Arguments[i].ID == id {
			return &u.Arguments[i]
		}
	}
	return &core.Argument{Title: id}
}

// render turns a ref into readable text: statements by their sentence,
// formulas recursively by their connective.
func render(u *core.Universe, ref string) string {
	for i := range u.Statements {
		if u.Statements[i].ID == ref {
			return u.Statements[i].Text
		}
	}
	for i := range u.Formulas {
		f := &u.Formulas[i]
		if f.ID != ref {
			continue
		}
		if f.Op == core.OpNot {
			return "NOT (" + render(u, f.Args[0]) + ")"
		}
		parts := make([]string, len(f.Args))
		for j, a := range f.Args {
			parts[j] = render(u, a)
		}
		return "(" + strings.Join(parts, " "+string(f.Op)+" ") + ")"
	}
	return ref
}
