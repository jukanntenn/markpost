import { describe, it, expect } from 'vitest'
import { compileKeywordFilter } from './keyword-filter'

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
