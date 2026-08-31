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
  | { type: 'or'; left: FilterNode; right: FilterNode }
  | { type: 'and'; left: FilterNode; right: FilterNode }
  | { type: 'not'; operand: FilterNode }
  | { type: 'keyword'; value: string }

type TokenKind =
  'eof' | 'comma' | 'pipe' | 'amp' | 'not' | 'lparen' | 'rparen' | 'keyword'

type Token = { kind: TokenKind; pos: number; value?: string }

function isOperatorByte(c: string): boolean {
  return (
    c === ',' ||
    c === '|' ||
    c === '&' ||
    c === '!' ||
    c === '(' ||
    c === ')' ||
    c === '"'
  )
}

export type FilterErrorCode =
  'unterminated_quote' | 'unexpected_token' | 'missing_rparen' | 'empty_keyword'

/**
 * Structured parse error: `code` selects the localized message, `pos` is the
 * 1-based code-point index of the offending character (null when the error is
 * "ran out of input" — no character to point at), `token` names the offending
 * token for unexpected_token/missing_rparen.
 */
export interface FilterError {
  code: FilterErrorCode
  pos: number | null
  token: TokenKind | null
  message: string
}

export class FilterParseError extends Error implements FilterError {
  readonly code: FilterErrorCode
  readonly pos: number | null
  readonly token: TokenKind | null

  constructor(
    input: string,
    unitPos: number,
    code: FilterErrorCode,
    token: TokenKind | null = null,
    message: string,
  ) {
    super(message)
    this.code = code
    this.token = token
    // eof has no character to point at — only real tokens carry a position.
    this.pos = token === 'eof' ? null : codePointPos(input, unitPos)
  }
}

// Token positions are UTF-16 code-unit indices; user-facing errors speak in
// code points so emoji count as one character.
function codePointPos(input: string, unitPos: number): number {
  return [...input.slice(0, unitPos)].length + 1
}

function tokenize(input: string): Token[] {
  const tokens: Token[] = []
  let pos = 0
  const isSpace = (s: string) => /\s/.test(s)

  const pushBare = (start: number, buf: string) => {
    tokens.push({ kind: 'keyword', pos: start, value: buf.trim() })
  }

  while (pos < input.length) {
    while (pos < input.length && isSpace(input[pos])) pos++
    if (pos >= input.length) break

    const c = input[pos]
    if (
      c === ',' ||
      c === '|' ||
      c === '&' ||
      c === '!' ||
      c === '(' ||
      c === ')'
    ) {
      tokens.push({ kind: OPERATOR_KINDS[c], pos })
      pos++
      continue
    }
    if (c === '"') {
      const start = pos
      pos++
      let buf = ''
      while (true) {
        if (pos >= input.length) {
          throw new FilterParseError(
            input,
            start,
            'unterminated_quote',
            null,
            'unterminated quoted string',
          )
        }
        const d = input[pos]
        if (d === '"') {
          if (pos + 1 < input.length && input[pos + 1] === '"') {
            buf += '"'
            pos += 2
            continue
          }
          pos++
          break
        }
        buf += d
        pos++
      }
      tokens.push({ kind: 'keyword', pos: start, value: buf })
      continue
    }

    const start = pos
    let buf = ''
    while (pos < input.length && !isOperatorByte(input[pos])) {
      buf += input[pos]
      pos++
    }
    pushBare(start, buf)
  }

  tokens.push({ kind: 'eof', pos: input.length })
  return tokens
}

const OPERATOR_KINDS: Record<string, TokenKind> = {
  ',': 'comma',
  '|': 'pipe',
  '&': 'amp',
  '!': 'not',
  '(': 'lparen',
  ')': 'rparen',
}

class Parser {
  private tokens: Token[]
  private cur = 0

  constructor(
    private input: string,
    tokens: Token[],
  ) {
    this.tokens = tokens
  }

  private peek(): Token {
    return this.tokens[this.cur]
  }

  private advance(): Token {
    return this.tokens[this.cur++]
  }

  parse(): FilterNode | null {
    if (this.peek().kind === 'eof') return null
    const node = this.parseOr()
    if (this.peek().kind !== 'eof') {
      const tok = this.peek()
      throw new FilterParseError(
        this.input,
        tok.pos,
        'unexpected_token',
        tok.kind,
        `unexpected ${tok.kind}`,
      )
    }
    return node
  }

  private parseOr(): FilterNode {
    let left = this.parseAnd()
    while (this.peek().kind === 'comma' || this.peek().kind === 'pipe') {
      this.advance()
      const right = this.parseAnd()
      left = { type: 'or', left, right }
    }
    return left
  }

  private parseAnd(): FilterNode {
    let left = this.parseNot()
    while (this.peek().kind === 'amp') {
      this.advance()
      const right = this.parseNot()
      left = { type: 'and', left, right }
    }
    return left
  }

  private parseNot(): FilterNode {
    if (this.peek().kind === 'not') {
      this.advance()
      return { type: 'not', operand: this.parseNot() }
    }
    return this.parseFactor()
  }

  private parseFactor(): FilterNode {
    const tok = this.peek()
    if (tok.kind === 'lparen') {
      this.advance()
      const inner = this.parseOr()
      if (this.peek().kind !== 'rparen') {
        const found = this.peek()
        throw new FilterParseError(
          this.input,
          found.pos,
          'missing_rparen',
          found.kind,
          `expected ')', got ${found.kind}`,
        )
      }
      this.advance()
      return inner
    }
    if (tok.kind === 'keyword') {
      // Token.value is optional at the type level (operators carry none), so
      // the empty-keyword guard must also narrow undefined away.
      if (tok.value === undefined || tok.value === '') {
        throw new FilterParseError(
          this.input,
          tok.pos,
          'empty_keyword',
          'keyword',
          'empty keyword',
        )
      }
      this.advance()
      return { type: 'keyword', value: tok.value }
    }
    throw new FilterParseError(
      this.input,
      tok.pos,
      'unexpected_token',
      tok.kind,
      `unexpected ${tok.kind}`,
    )
  }
}

export interface CompileResult {
  node: FilterNode | null
  error: FilterError | null
}

export function compileKeywordFilter(expr: string): CompileResult {
  try {
    const tokens = tokenize(expr)
    const node = new Parser(expr, tokens).parse()
    return { node, error: null }
  } catch (e) {
    if (e instanceof FilterParseError) {
      return {
        node: null,
        error: { code: e.code, pos: e.pos, token: e.token, message: e.message },
      }
    }
    return {
      node: null,
      error: {
        code: 'unexpected_token',
        pos: null,
        token: null,
        message: 'parse failed',
      },
    }
  }
}

/**
 * Locale phrases needed to render an AST as a natural-language sentence.
 * Each locale provides the connectives; the walker below owns structure
 * (precedence-aware parentheses, double-negation folding).
 */
export interface Phrasebook {
  quote: (kw: string) => string
  contains: (kw: string) => string
  notContains: (kw: string) => string
  notGroup: (inner: string) => string
  and: (a: string, b: string) => string
  or: (a: string, b: string) => string
  group: (inner: string) => string
}

const MAX_KEYWORD_DISPLAY = 30

function displayKeyword(value: string): string {
  const cps = [...value]
  if (cps.length <= MAX_KEYWORD_DISPLAY) return value
  return cps.slice(0, MAX_KEYWORD_DISPLAY - 1).join('') + '…'
}

/**
 * Render a parsed filter as a natural-language clause describing the title
 * condition, e.g. 包含「prod」且（包含「error」或「warning」）且不包含「debug」.
 * Returns null for the empty expression (matches everything — the caller
 * shows its own "deliver all" phrase).
 */
export function describeFilter(
  node: FilterNode | null,
  phr: Phrasebook,
): string | null {
  if (node === null) return null

  const render = (n: FilterNode): string => {
    switch (n.type) {
      case 'keyword':
        return phr.contains(phr.quote(displayKeyword(n.value)))
      case 'not': {
        // Fold double negation (!!a === a) before rendering.
        let inner = n.operand
        if (inner.type === 'not') inner = inner.operand
        return inner.type === 'keyword'
          ? phr.notContains(phr.quote(displayKeyword(inner.value)))
          : phr.notGroup(render(inner))
      }
      case 'and':
        return phr.and(
          n.left.type === 'or' ? phr.group(render(n.left)) : render(n.left),
          n.right.type === 'or' ? phr.group(render(n.right)) : render(n.right),
        )
      case 'or':
        return phr.or(
          n.left.type === 'and' ? phr.group(render(n.left)) : render(n.left),
          n.right.type === 'and' ? phr.group(render(n.right)) : render(n.right),
        )
    }
  }

  return render(node)
}
