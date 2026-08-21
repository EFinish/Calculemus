// Hand-written mirror of core's JSON shapes (core/model.go, core/queries.go).
// Deliberately not code-generated — leibniz died in protobuf codegen churn;
// this file is ~80 lines and changes when the Go model does. Keep the two in
// sight of each other.

export type Quantifier = "ALL" | "SOME" | "NONE";
export type Qualifier = "IS" | "IS_NOT";

export type Op =
  | "NOT"
  | "AND"
  | "OR"
  | "IMPLIES"
  | "IFF"
  | "XOR"
  | "NAND"
  | "NOR"
  | "XNOR";

export const UNARY_OPS: Op[] = ["NOT"];
export const BINARY_OPS: Op[] = ["IMPLIES", "IFF", "XOR", "XNOR"];
export const NARY_OPS: Op[] = ["AND", "OR", "NAND", "NOR"];
export const ALL_OPS: Op[] = [...UNARY_OPS, ...BINARY_OPS, ...NARY_OPS];

export interface Statement {
  id: string;
  text: string;
  subject?: string;
  quantifier?: Quantifier;
  predicate?: string;
  qualifier?: Qualifier;
  // M6 relational grammar (DESIGN §11): verb "" or absent = copula.
  verb?: string;
  subjectIsIndividual?: boolean;
  objectIsIndividual?: boolean;
  objectQuantifier?: Quantifier;
}

export interface Formula {
  id: string;
  op: Op;
  args: string[];
}

export interface Assertion {
  formula: string;
  active: boolean;
  source?: string;
}

export interface Argument {
  id: string;
  title: string;
  premises: string[];
  conclusion: string;
}

export interface Scenario {
  name: string;
  toggles: Record<string, boolean>;
}

export interface Point {
  x: number;
  y: number;
}

export interface Universe {
  version: number;
  title: string;
  witnesses?: number;
  statements: Statement[];
  formulas?: Formula[];
  assertions?: Assertion[];
  arguments?: Argument[];
  scenarios?: Scenario[];
  layout?: Record<string, Point>;
}

export type EdgeType = "shares" | "contradicts" | "chains";

export interface Edge {
  type: EdgeType;
  from: string;
  to: string;
}

export interface ArgumentVerdict {
  id: string;
  valid: boolean;
  form?: string;
  countermodel?: Record<string, boolean>;
}

export interface Verdicts {
  consistent: boolean;
  unsatCore?: string[];
  entailedTrue?: string[];
  entailedFalse?: string[];
  vacuous?: string[];
  arguments?: ArgumentVerdict[];
  edges?: Edge[];
  boundedDomain?: number;
}
