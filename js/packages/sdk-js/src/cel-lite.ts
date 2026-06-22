// cel-lite — a tiny subset of CEL good enough for the slice 2
// fixtures. Supports: dot-access identifiers (ctx.user.country),
// boolean (&&, ||, !), comparisons (==, !=, <, <=, >, >=), the `in`
// membership operator with list literals, string/number/bool/null
// literals, and parentheses.
//
// Strings may be single- or double-quoted. Numbers are JS numbers
// (no BigInt). The grammar is intentionally simple; if a fixture
// ever needs something richer (string functions, arithmetic, etc.)
// extend this AND the Go evaluator together so the cross-runtime
// corpus stays the source of truth.

// ---- Lexer ---------------------------------------------------------

type Token =
  | { kind: "string"; value: string }
  | { kind: "number"; value: number }
  | { kind: "bool"; value: boolean }
  | { kind: "null" }
  | { kind: "ident"; value: string }
  | { kind: "punct"; value: string }
  | { kind: "eof" };

function tokenize(src: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  const len = src.length;
  while (i < len) {
    const ch = src.charAt(i);
    // whitespace
    if (ch === " " || ch === "\t" || ch === "\n" || ch === "\r") {
      i++;
      continue;
    }
    // string literal
    if (ch === "'" || ch === '"') {
      const quote = ch;
      let end = i + 1;
      let out = "";
      while (end < len && src[end] !== quote) {
        if (src[end] === "\\" && end + 1 < len) {
          out += src[end + 1];
          end += 2;
          continue;
        }
        out += src[end];
        end++;
      }
      if (end >= len) throw new Error(`cel-lite: unterminated string at ${i}`);
      tokens.push({ kind: "string", value: out });
      i = end + 1;
      continue;
    }
    // number literal (integer or float)
    if (/[0-9]/.test(ch)) {
      let end = i;
      while (end < len && /[0-9.]/.test(src.charAt(end))) end++;
      tokens.push({ kind: "number", value: Number(src.slice(i, end)) });
      i = end;
      continue;
    }
    // identifier or keyword
    if (/[A-Za-z_]/.test(ch)) {
      let end = i;
      while (end < len && /[A-Za-z0-9_]/.test(src.charAt(end))) end++;
      const word = src.slice(i, end);
      if (word === "true" || word === "false") {
        tokens.push({ kind: "bool", value: word === "true" });
      } else if (word === "null") {
        tokens.push({ kind: "null" });
      } else if (word === "in") {
        tokens.push({ kind: "punct", value: "in" });
      } else {
        tokens.push({ kind: "ident", value: word });
      }
      i = end;
      continue;
    }
    // multi-char punctuation
    if (i + 1 < len) {
      const two = src.slice(i, i + 2);
      if (
        two === "==" ||
        two === "!=" ||
        two === "<=" ||
        two === ">=" ||
        two === "&&" ||
        two === "||"
      ) {
        tokens.push({ kind: "punct", value: two });
        i += 2;
        continue;
      }
    }
    // single-char punctuation
    if ("().<>!,[].".includes(ch)) {
      tokens.push({ kind: "punct", value: ch });
      i++;
      continue;
    }
    throw new Error(
      `cel-lite: unexpected character ${JSON.stringify(ch)} at ${i}`,
    );
  }
  tokens.push({ kind: "eof" });
  return tokens;
}

// ---- AST -----------------------------------------------------------

type Expr =
  | { type: "literal"; value: unknown }
  | { type: "path"; parts: string[] }
  | { type: "list"; items: Expr[] }
  | { type: "unary"; op: "!"; operand: Expr }
  | { type: "binary"; op: BinOp; left: Expr; right: Expr };

type BinOp = "==" | "!=" | "<" | "<=" | ">" | ">=" | "&&" | "||" | "in";

// ---- Parser --------------------------------------------------------

class Parser {
  private pos = 0;
  constructor(private tokens: Token[]) {}

  private peek(): Token {
    const t = this.tokens[this.pos];
    if (!t) throw new Error("cel-lite: peek past end of input");
    return t;
  }
  private next(): Token {
    const t = this.tokens[this.pos++];
    if (!t) throw new Error("cel-lite: next past end of input");
    return t;
  }
  private acceptPunct(value: string): boolean {
    const t = this.peek();
    if (t.kind === "punct" && t.value === value) {
      this.pos++;
      return true;
    }
    return false;
  }
  private expectPunct(value: string): void {
    if (!this.acceptPunct(value)) {
      throw new Error(
        `cel-lite: expected ${value}, got ${JSON.stringify(this.peek())}`,
      );
    }
  }

  parse(): Expr {
    const expr = this.parseOr();
    if (this.peek().kind !== "eof") {
      throw new Error(`cel-lite: trailing tokens at ${this.pos}`);
    }
    return expr;
  }

  private parseOr(): Expr {
    let left = this.parseAnd();
    while (this.acceptPunct("||")) {
      const right = this.parseAnd();
      left = { type: "binary", op: "||", left, right };
    }
    return left;
  }

  private parseAnd(): Expr {
    let left = this.parseNot();
    while (this.acceptPunct("&&")) {
      const right = this.parseNot();
      left = { type: "binary", op: "&&", left, right };
    }
    return left;
  }

  private parseNot(): Expr {
    if (this.acceptPunct("!")) {
      return { type: "unary", op: "!", operand: this.parseNot() };
    }
    return this.parseRel();
  }

  private parseRel(): Expr {
    const left = this.parsePrimary();
    const op = this.peek();
    if (
      op.kind === "punct" &&
      (op.value === "==" ||
        op.value === "!=" ||
        op.value === "<" ||
        op.value === "<=" ||
        op.value === ">" ||
        op.value === ">=" ||
        op.value === "in")
    ) {
      this.pos++;
      const right = this.parsePrimary();
      return { type: "binary", op: op.value as BinOp, left, right };
    }
    return left;
  }

  private parsePrimary(): Expr {
    const t = this.peek();
    if (t.kind === "punct" && t.value === "(") {
      this.pos++;
      const inner = this.parseOr();
      this.expectPunct(")");
      return inner;
    }
    if (t.kind === "punct" && t.value === "[") {
      this.pos++;
      const items: Expr[] = [];
      if (!this.acceptPunct("]")) {
        items.push(this.parseOr());
        while (this.acceptPunct(",")) items.push(this.parseOr());
        this.expectPunct("]");
      }
      return { type: "list", items };
    }
    if (t.kind === "string") {
      this.pos++;
      return { type: "literal", value: t.value };
    }
    if (t.kind === "number") {
      this.pos++;
      return { type: "literal", value: t.value };
    }
    if (t.kind === "bool") {
      this.pos++;
      return { type: "literal", value: t.value };
    }
    if (t.kind === "null") {
      this.pos++;
      return { type: "literal", value: null };
    }
    if (t.kind === "ident") {
      const parts: string[] = [];
      parts.push((this.next() as { value: string }).value);
      while (this.acceptPunct(".")) {
        const seg = this.next();
        if (seg.kind !== "ident")
          throw new Error(`cel-lite: expected ident after '.'`);
        parts.push(seg.value);
      }
      return { type: "path", parts };
    }
    throw new Error(`cel-lite: unexpected token ${JSON.stringify(t)}`);
  }
}

// ---- Evaluator -----------------------------------------------------

export interface CelProgram {
  source: string;
  ast: Expr;
}

export function compileCEL(source: string): CelProgram {
  const tokens = tokenize(source);
  const ast = new Parser(tokens).parse();
  return { source, ast };
}

export function evalCEL(
  prog: CelProgram,
  ctx: Record<string, unknown>,
): boolean {
  const v = evalNode(prog.ast, { ctx });
  return v === true;
}

function evalNode(node: Expr, env: Record<string, unknown>): unknown {
  switch (node.type) {
    case "literal":
      return node.value;
    case "path":
      return lookupPath(env, node.parts);
    case "list":
      return node.items.map((n) => evalNode(n, env));
    case "unary":
      return !truthy(evalNode(node.operand, env));
    case "binary": {
      // Short-circuit boolean operators evaluate lazily.
      if (node.op === "&&") {
        const l = evalNode(node.left, env);
        if (!truthy(l)) return false;
        return truthy(evalNode(node.right, env));
      }
      if (node.op === "||") {
        const l = evalNode(node.left, env);
        if (truthy(l)) return true;
        return truthy(evalNode(node.right, env));
      }
      const l = evalNode(node.left, env);
      const r = evalNode(node.right, env);
      switch (node.op) {
        case "==":
          return jsonEqual(l, r);
        case "!=":
          return !jsonEqual(l, r);
        case "<":
          return num(l) < num(r);
        case "<=":
          return num(l) <= num(r);
        case ">":
          return num(l) > num(r);
        case ">=":
          return num(l) >= num(r);
        case "in":
          return Array.isArray(r) && r.some((item) => jsonEqual(item, l));
      }
    }
  }
}

function lookupPath(root: Record<string, unknown>, parts: string[]): unknown {
  let cur: unknown = root;
  for (const p of parts) {
    if (cur === null || cur === undefined || typeof cur !== "object")
      return undefined;
    cur = (cur as Record<string, unknown>)[p];
  }
  return cur;
}

function truthy(v: unknown): boolean {
  return v === true;
}

function num(v: unknown): number {
  if (typeof v === "number") return v;
  if (typeof v === "bigint") return Number(v);
  return Number.NaN;
}

function jsonEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (a === null || b === null) return a === b;
  if (typeof a !== typeof b) return false;
  if (typeof a === "object") {
    return JSON.stringify(a) === JSON.stringify(b);
  }
  return false;
}
