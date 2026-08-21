// Package core is the Calculemus engine: the universe data model, a small
// SAT solver, and the semantic queries (consistency, validity, entailment,
// diagnosis) that every product feature is built on.
//
// core has zero dependencies and never imports from wasm/, app/, or server/.
// The one intended boundary for callers is Evaluate / EvaluateScenario:
// universe in, verdicts out.
package core

// Quantifier and Qualifier carry apriorio's term structure. They are stored
// from day one (guardrail R5) but have no semantics until milestone M4;
// until then a Statement is an opaque propositional atom.
type Quantifier string

const (
	QuantAll  Quantifier = "ALL"
	QuantSome Quantifier = "SOME"
	QuantNone Quantifier = "NONE"
)

type Qualifier string

const (
	QualIs    Qualifier = "IS"
	QualIsNot Qualifier = "IS_NOT"
)

// Statement is the atom: a declarative sentence the engine treats as a
// propositional variable (until term structure gives it internal semantics).
//
// M6 (DESIGN §11) adds the relational form: subject phrase + verb + object
// phrase. Verb "" means the copula, which is exactly the M4 shape — so every
// pre-M6 document parses and means the same thing. Predicate doubles as the
// object name for verb statements. A phrase is either an *individual* (a
// constant: "the boy") or a quantified *kind* ("all men").
type Statement struct {
	ID         string     `json:"id"`
	Text       string     `json:"text"`
	Subject    string     `json:"subject,omitempty"`
	Quantifier Quantifier `json:"quantifier,omitempty"`
	Predicate  string     `json:"predicate,omitempty"`
	Qualifier  Qualifier  `json:"qualifier,omitempty"` // IS/IS_NOT; for verbs, does/does not

	Verb                string     `json:"verb,omitempty"`
	SubjectIsIndividual bool       `json:"subjectIsIndividual,omitempty"`
	ObjectIsIndividual  bool       `json:"objectIsIndividual,omitempty"`
	ObjectQuantifier    Quantifier `json:"objectQuantifier,omitempty"` // kind objects of verbs
}

// Op is a logical connective. XOR, NAND, NOR, XNOR are kept as sugar for
// continuity with apriorio; NOT and IMPLIES are first-class (guardrail R6).
type Op string

const (
	OpNot     Op = "NOT"     // arity 1
	OpAnd     Op = "AND"     // arity >= 2
	OpOr      Op = "OR"      // arity >= 2
	OpImplies Op = "IMPLIES" // arity 2: args[0] -> args[1]
	OpIff     Op = "IFF"     // arity 2
	OpXor     Op = "XOR"     // arity 2 (n-ary XOR is ambiguous: parity vs exactly-one)
	OpNand    Op = "NAND"    // arity >= 2
	OpNor     Op = "NOR"     // arity >= 2
	OpXnor    Op = "XNOR"    // arity 2
)

// Formula is a connective over refs. A ref is the id of a Statement or of
// another Formula; statements and formulas share one id namespace. Formulas
// must form a DAG (checked by Validate).
type Formula struct {
	ID   string   `json:"id"`
	Op   Op       `json:"op"`
	Args []string `json:"args"`
}

// Assertion commits to a formula (or statement) being true. Asserting
// falsehood is expressed by asserting a NOT formula.
type Assertion struct {
	Formula string `json:"formula"`
	Active  bool   `json:"active"`
	Source  string `json:"source,omitempty"` // "hand" or "argument:<id>"
}

// Argument is premises ⊢ conclusion. Premise is a role refs play here, not a
// type of thing. Validity is computed, never declared.
type Argument struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Premises   []string `json:"premises"`
	Conclusion string   `json:"conclusion"`
}

// Scenario is a named counterfactual: per-ref overrides of assertion
// activity. A toggle for a ref with no existing assertion adds one (true) or
// is a no-op (false).
type Scenario struct {
	Name    string          `json:"name"`
	Toggles map[string]bool `json:"toggles"`
}

// Point is canvas layout state. The engine never reads it; it lives in the
// document so one file round-trips the whole universe.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Universe is the document: one universe = one JSON file = one canvas.
type Universe struct {
	Version int    `json:"version"`
	Title   string `json:"title"`
	// Witnesses: anonymous domain elements added beyond the named individuals
	// when relational semantics are in play (0 means the default, 4). See
	// DESIGN §11 — "valid" then means "no countermodel with ≤ N things".
	Witnesses  int              `json:"witnesses,omitempty"`
	Statements []Statement      `json:"statements"`
	Formulas   []Formula        `json:"formulas,omitempty"`
	Assertions []Assertion      `json:"assertions,omitempty"`
	Arguments  []Argument       `json:"arguments,omitempty"`
	Scenarios  []Scenario       `json:"scenarios,omitempty"`
	Layout     map[string]Point `json:"layout,omitempty"`
}
