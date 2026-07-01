/**
 * Keyword filter expression — frontend port of the Go backend grammar
 * (backend/internal/service/delivery/filter). The backend is authoritative;
 * this module mirrors the syntax so the form can validate and preview the
 * expression client-side.
 *
 * Grammar (Model 2: spaces are keyword content, not separators):
 *   expr   := or
 *   or     := and  ( ("," | "|") and )*   // OR, lowest precedence
 *   and    := not  ( "&" not )*           // AND
 *   not    := "!" not | factor            // NOT, prefix
 *   factor := KEYWORD | "(" expr ")"
 *
 * Operators are exactly seven ASCII chars: , | & ! ( ) "
 * Every other character is literal keyword content.
 */

export type FilterNode =
  | { type: "or"; left: FilterNode; right: FilterNode }
  | { type: "and"; left: FilterNode; right: FilterNode }
  | { type: "not"; operand: FilterNode }
  | { type: "keyword"; value: string };

type Token =
  | { kind: "eof" }
  | { kind: "comma" }
  | { kind: "pipe" }
  | { kind: "amp" }
  | { kind: "not" }
  | { kind: "lparen" }
  | { kind: "rparen" }
  | { kind: "keyword"; value: string };

function isOperatorByte(c: string): boolean {
  return c === "," || c === "|" || c === "&" || c === "!" || c === "(" || c === ")" || c === '"';
}

function tokenize(input: string): Token[] {
  const tokens: Token[] = [];
  let pos = 0;
  const isSpace = (s: string) => /\s/.test(s);

  const pushBare = (buf: string) => {
    const value = buf.trim();
    tokens.push({ kind: "keyword", value });
  };

  while (pos < input.length) {
    while (pos < input.length && isSpace(input[pos])) pos++;
    if (pos >= input.length) break;

    const c = input[pos];
    if (c === ",") { tokens.push({ kind: "comma" }); pos++; continue; }
    if (c === "|") { tokens.push({ kind: "pipe" }); pos++; continue; }
    if (c === "&") { tokens.push({ kind: "amp" }); pos++; continue; }
    if (c === "!") { tokens.push({ kind: "not" }); pos++; continue; }
    if (c === "(") { tokens.push({ kind: "lparen" }); pos++; continue; }
    if (c === ")") { tokens.push({ kind: "rparen" }); pos++; continue; }
    if (c === '"') {
      pos++;
      let buf = "";
      while (true) {
        if (pos >= input.length) {
          throw new FilterParseError("unterminated quoted string");
        }
        const d = input[pos];
        if (d === '"') {
          if (pos + 1 < input.length && input[pos + 1] === '"') {
            buf += '"';
            pos += 2;
            continue;
          }
          pos++;
          break;
        }
        buf += d;
        pos++;
      }
      tokens.push({ kind: "keyword", value: buf });
      continue;
    }

    let buf = "";
    while (pos < input.length && !isOperatorByte(input[pos])) {
      buf += input[pos];
      pos++;
    }
    pushBare(buf);
  }

  tokens.push({ kind: "eof" });
  return tokens;
}

export class FilterParseError extends Error {}

class Parser {
  private tokens: Token[];
  private cur = 0;

  constructor(tokens: Token[]) {
    this.tokens = tokens;
  }

  private peek(): Token {
    return this.tokens[this.cur];
  }

  private advance(): Token {
    return this.tokens[this.cur++];
  }

  parse(): FilterNode | null {
    if (this.peek().kind === "eof") return null;
    const node = this.parseOr();
    if (this.peek().kind !== "eof") {
      throw new FilterParseError(`unexpected ${this.peek().kind}`);
    }
    return node;
  }

  private parseOr(): FilterNode {
    let left = this.parseAnd();
    while (this.peek().kind === "comma" || this.peek().kind === "pipe") {
      this.advance();
      const right = this.parseAnd();
      left = { type: "or", left, right };
    }
    return left;
  }

  private parseAnd(): FilterNode {
    let left = this.parseNot();
    while (this.peek().kind === "amp") {
      this.advance();
      const right = this.parseNot();
      left = { type: "and", left, right };
    }
    return left;
  }

  private parseNot(): FilterNode {
    if (this.peek().kind === "not") {
      this.advance();
      return { type: "not", operand: this.parseNot() };
    }
    return this.parseFactor();
  }

  private parseFactor(): FilterNode {
    const tok = this.peek();
    if (tok.kind === "lparen") {
      this.advance();
      const inner = this.parseOr();
      if (this.peek().kind !== "rparen") {
        throw new FilterParseError(`expected ')', got ${this.peek().kind}`);
      }
      this.advance();
      return inner;
    }
    if (tok.kind === "keyword") {
      if (tok.value === "") {
        throw new FilterParseError("empty keyword");
      }
      this.advance();
      return { type: "keyword", value: tok.value };
    }
    throw new FilterParseError(`unexpected ${tok.kind}`);
  }
}

export interface CompileResult {
  node: FilterNode | null;
  error: string | null;
}

export function compileKeywordFilter(expr: string): CompileResult {
  try {
    const tokens = tokenize(expr);
    const node = new Parser(tokens).parse();
    return { node, error: null };
  } catch (e) {
    if (e instanceof FilterParseError) {
      return { node: null, error: e.message };
    }
    return { node: null, error: "parse failed" };
  }
}

/** Quote a keyword value for display inside the human-readable description. */
function displayKeyword(value: string): string {
  if (/[,|&!()" ]/.test(value) || value === "") {
    return `"${value}"`;
  }
  return value;
}

/**
 * Render a compiled node as a human-readable description. Parentheses are
 * inserted only where needed to make the precedence unambiguous to a reader
 * (an AND under an OR, or an AND/OR under a NOT). Returns null when the
 * expression is empty (matches everything).
 */
export function describeFilter(node: FilterNode | null): string | null {
  if (node === null) return null;
  const walk = (n: FilterNode): string => {
    switch (n.type) {
      case "or":
        return `${parenIfAnd(n.left)} | ${parenIfAnd(n.right)}`;
      case "and":
        return `${walk(n.left)} & ${walk(n.right)}`;
      case "not":
        return `!${parenIfCompound(n.operand)}`;
      case "keyword":
        return displayKeyword(n.value);
    }
  };
  const parenIfAnd = (n: FilterNode): string =>
    n.type === "and" ? `(${walk(n)})` : walk(n);
  const parenIfCompound = (n: FilterNode): string =>
    n.type === "and" || n.type === "or" ? `(${walk(n)})` : walk(n);
  return walk(node);
}
