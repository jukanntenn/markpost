import { describe, it, expect } from 'vitest'
import enMessages from '../i18n/locales/en.json'
import zhHansMessages from '../i18n/locales/zh-Hans.json'
import {
  compileKeywordFilter,
  describeFilter,
  type Phrasebook,
} from './keyword-filter'

function phrasebookFrom(locale: typeof enMessages): Phrasebook {
  const m = (
    locale as unknown as {
      delivery: { dialog: Record<string, string> }
    }
  ).delivery.dialog
  const fmt = (tpl: string, params: Record<string, string>) =>
    tpl.replace(/\{(\w+)\}/g, (_, k: string) => params[k] ?? '')
  return {
    quote: (kw) => fmt(m.keywordsPhQuote, { kw }),
    contains: (kw) => fmt(m.keywordsPhContains, { kw }),
    notContains: (kw) => fmt(m.keywordsPhNotContains, { kw }),
    notGroup: (inner) => fmt(m.keywordsPhNotGroup, { inner }),
    and: (a, b) => fmt(m.keywordsPhAnd, { a, b }),
    or: (a, b) => fmt(m.keywordsPhOr, { a, b }),
    group: (inner) => fmt(m.keywordsPhGroup, { inner }),
  }
}

describe('compileKeywordFilter', () => {
  it('returns null node for empty input', () => {
    expect(compileKeywordFilter('').node).toBeNull()
    expect(compileKeywordFilter('   ').node).toBeNull()
  })

  it('parses single keyword', () => {
    const { node, error } = compileKeywordFilter('alpha')
    expect(error).toBeNull()
    expect(node).toEqual({ type: 'keyword', value: 'alpha' })
  })

  it('parses OR with comma and pipe equivalently', () => {
    expect(compileKeywordFilter('a, b, c').node).toEqual({
      type: 'or',
      left: {
        type: 'or',
        left: { type: 'keyword', value: 'a' },
        right: { type: 'keyword', value: 'b' },
      },
      right: { type: 'keyword', value: 'c' },
    })
    const pipe = compileKeywordFilter('a | b | c')
    const comma = compileKeywordFilter('a, b, c')
    expect(JSON.stringify(pipe.node)).toBe(JSON.stringify(comma.node))
  })

  it('parses AND', () => {
    expect(compileKeywordFilter('a & b & c').node).toEqual({
      type: 'and',
      left: {
        type: 'and',
        left: { type: 'keyword', value: 'a' },
        right: { type: 'keyword', value: 'b' },
      },
      right: { type: 'keyword', value: 'c' },
    })
  })

  it('parses NOT and double negation', () => {
    expect(compileKeywordFilter('!a').node).toEqual({
      type: 'not',
      operand: { type: 'keyword', value: 'a' },
    })
    expect(compileKeywordFilter('!!a').node).toEqual({
      type: 'not',
      operand: { type: 'not', operand: { type: 'keyword', value: 'a' } },
    })
  })

  it('treats spaces as keyword content (Model 2)', () => {
    expect(compileKeywordFilter('key word 1').node).toEqual({
      type: 'keyword',
      value: 'key word 1',
    })
  })

  it('parses parentheses and nested grouping', () => {
    expect(compileKeywordFilter('(a | b) & c').node).toEqual({
      type: 'and',
      left: {
        type: 'or',
        left: { type: 'keyword', value: 'a' },
        right: { type: 'keyword', value: 'b' },
      },
      right: { type: 'keyword', value: 'c' },
    })
    expect(compileKeywordFilter('((a | b) & !c) | d').error).toBeNull()
  })

  it('handles quoted keywords with operator characters', () => {
    expect(compileKeywordFilter(`"a,b"`).node).toEqual({
      type: 'keyword',
      value: 'a,b',
    })
    expect(compileKeywordFilter(`"a & b"`).node).toEqual({
      type: 'keyword',
      value: 'a & b',
    })
  })

  it('handles double-quote doubling', () => {
    expect(compileKeywordFilter(`"say ""hi"""`).node).toEqual({
      type: 'keyword',
      value: `say "hi"`,
    })
    expect(compileKeywordFilter(`""""`).node).toEqual({
      type: 'keyword',
      value: `"`,
    })
  })

  it('rejects empty keyword and unterminated quote', () => {
    expect(compileKeywordFilter(`""`).error).not.toBeNull()
    expect(compileKeywordFilter(`"abc`).error).not.toBeNull()
  })

  it('rejects structural errors', () => {
    const invalid = [
      'a,,b',
      'a && b',
      'a &',
      '& a',
      '&',
      '|',
      ',',
      ',a',
      'a,',
      '!',
      '(a',
      'a)',
      '()',
      '(a,)',
      'a (b)',
      '(a)(b)',
      `a"b"`,
      'a & , b',
    ]
    for (const expr of invalid) {
      expect(compileKeywordFilter(expr).error, `expr=${expr}`).not.toBeNull()
    }
  })
})

describe('structured parse errors', () => {
  it('reports code, offending token, and 1-based position', () => {
    expect(compileKeywordFilter('"abc').error).toMatchObject({
      code: 'unterminated_quote',
      pos: 1,
    })
    expect(compileKeywordFilter('a)').error).toMatchObject({
      code: 'unexpected_token',
      token: 'rparen',
      pos: 2,
    })
    expect(compileKeywordFilter('a,,b').error).toMatchObject({
      code: 'unexpected_token',
      token: 'comma',
      pos: 3,
    })
    expect(compileKeywordFilter('""').error).toMatchObject({
      code: 'empty_keyword',
      pos: 1,
    })
    expect(compileKeywordFilter('(a').error).toMatchObject({
      code: 'missing_rparen',
      token: 'eof',
    })
    expect(compileKeywordFilter('a &').error).toMatchObject({
      code: 'unexpected_token',
      token: 'eof',
      pos: null,
    })
  })

  it('counts astral characters (emoji) as one character', () => {
    expect(compileKeywordFilter('🚀)').error).toMatchObject({
      code: 'unexpected_token',
      token: 'rparen',
      pos: 2,
    })
  })
})

describe('describeFilter', () => {
  const en = phrasebookFrom(enMessages)
  const zh = phrasebookFrom(zhHansMessages)
  const node = (expr: string) => compileKeywordFilter(expr).node

  it('returns null for the empty expression', () => {
    expect(describeFilter(null, en)).toBeNull()
  })

  it('renders a single keyword', () => {
    expect(describeFilter(node('alert'), en)).toBe('contains “alert”')
    expect(describeFilter(node('alert'), zh)).toBe('包含「alert」')
  })

  it('renders OR chains', () => {
    expect(describeFilter(node('mark, post'), en)).toBe(
      'contains “mark” or contains “post”',
    )
    expect(describeFilter(node('mark, post'), zh)).toBe(
      '包含「mark」或包含「post」',
    )
  })

  it('wraps nested OR inside AND with parentheses', () => {
    expect(describeFilter(node('(error, warning) & prod'), en)).toBe(
      '(contains “error” or contains “warning”) and contains “prod”',
    )
    expect(describeFilter(node('prod & (error, warning) & !debug'), zh)).toBe(
      '包含「prod」且（包含「error」或包含「warning」）且不包含「debug」',
    )
  })

  it('wraps nested AND inside OR with parentheses', () => {
    expect(describeFilter(node('a | b & c'), en)).toBe(
      'contains “a” or (contains “b” and contains “c”)',
    )
  })

  it('renders NOT on keywords and groups', () => {
    expect(describeFilter(node('!debug'), en)).toBe('does not contain “debug”')
    expect(describeFilter(node('!(a, b)'), en)).toBe(
      'does not satisfy (contains “a” or contains “b”)',
    )
    expect(describeFilter(node('!(a, b)'), zh)).toBe(
      '不满足（包含「a」或包含「b」）',
    )
  })

  it('folds double negation', () => {
    expect(describeFilter(node('!!a'), en)).toBe('does not contain “a”')
  })

  it('truncates overlong keywords with an ellipsis', () => {
    expect(describeFilter(node('x'.repeat(40)), en)).toBe(
      `contains “${'x'.repeat(29)}…”`,
    )
  })
})
