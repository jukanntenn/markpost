# Keyword Filter Expressions

This document specifies the keyword filter expression grammar that delivery channels use to decide whether a post should be pushed. It is the authoritative reference for the syntax, matching semantics, validation rules, and the cross-language contract between the Go backend (evaluator) and the TypeScript frontend (live preview).

## Overview

Each delivery channel has a `keywords` text field holding a **filter expression**. When a post is delivered, the expression is evaluated against the post **title** (substring matching). If the title satisfies the expression, the post is pushed to that channel; otherwise it is skipped. An empty expression matches every post (always deliver).

The expression language is a standard boolean algebra (OR / AND / NOT with parentheses), designed so that:

- Common cases need zero learning: `a, b, c` (OR), `a & b` (AND), `!a` (NOT).
- Spaces are part of keyword content — multi-word phrases like `key word 1` need **no quotes**.
- There are exactly seven operator characters; every other character is literal keyword content.
- Malformed expressions are **rejected at write time**, never silently accepted. There is no way for a stored expression to produce an ambiguous or surprising match.

## Grammar

```
expr   := or
or     := and  ( ("," | "|") and )*      # OR — lowest precedence, left-associative
and    := not  ( "&" not )*              # AND — left-associative
not    := "!" not | factor               # NOT — prefix, right-associative
factor := KEYWORD | "(" expr ")"         # terminal or grouped sub-expression
```

**Precedence** (tightest to loosest): `!` > `&` > `,`/`|`. Parentheses override. This is the universally-familiar boolean precedence — no custom rules.

`a | b & c` therefore parses as `a | (b & c)`, and `!a & b` as `(!a) & b`.

## Operators

Exactly seven ASCII characters are operators:

| Char | Meaning | Notes |
|------|---------|-------|
| `,` | OR | Equivalent to `\|` |
| `\|` | OR | Equivalent to `,` |
| `&` | AND | |
| `!` | NOT (prefix) | Right-associative; `!!a == a`, `!!!a == !a` |
| `(` `)` | Grouping | Overrides precedence; may nest arbitrarily |
| `"` | Quoting | Makes operator characters literal; see below |

**Every other character is literal keyword content.** This includes letters, digits, CJK, emoji, and all punctuation except the seven above: `+ / \ : ; @ # $ % ^ = ~ [ ] { } < > ? * '` and so on. Such keywords need no quotes.

## Lexing Rules

### Keywords (Model 2: spaces are content)

An unquoted keyword is the **longest contiguous run of characters that does not contain any of the seven operator bytes**. After reading it, leading and trailing whitespace is trimmed (including U+3000 ideographic space). Internal whitespace is preserved and participates in matching.

- `key word 1` → one keyword `key word 1`.
- `C++`, `a/b`, `a\b`, `🚀go`, `错误` → single keywords, no quotes needed.
- `a & b` → keyword `a`, operator `&`, keyword `b`.

### Quoting `"..."`

A quoted segment treats everything between the quotes as literal content — all seven operators lose their meaning inside quotes. Leading and trailing spaces inside quotes are **preserved**.

- `"a,b"` → keyword `a,b`.
- `"a & b"` → keyword `a & b`.
- `" error "` → keyword ` error ` (leading and trailing space kept).

A literal double-quote inside a quoted segment is written by **doubling**: `""`.

- `"say ""hi"""` → keyword `say "hi"`.
- `""""` → keyword `"` (a single double-quote).

The backslash `\` is **always literal** (there is no backslash escaping). `a\b` is the keyword `a\b`, even inside quotes.

### When quotes are required

Quotes are required only when a keyword contains any of `, | & ! ( ) "` or when you want to preserve its leading/trailing whitespace. Otherwise quotes are optional — `"key word"` and `key word` are identical.

### Whitespace

Whitespace around operators is ignored: `a & b` ≡ `a&b`. Two adjacent factors with **no operator between them** are a syntax error (see *Rejections* below) — the only way to combine factors is with an explicit operator. This is the key property that makes the grammar unambiguous.

## Matching Semantics

For each keyword, matching is **case-insensitive substring** comparison against the title:

```
keyword'  = strings.ToLower(norm.NFC(keyword))
title'    = strings.ToLower(norm.NFC(title))
match     = strings.Contains(title', keyword')
```

| Dimension | Rule |
|-----------|------|
| Match type | Substring (quoted and unquoted are both substring) |
| Case | Insensitive — Unicode default case folding via `strings.ToLower` |
| Normalization | Both keyword and title normalized to **Unicode NFC** before comparison |
| Matched field | **Title only** (`post.DeliveryJob.Title`) |
| Empty / whitespace-only expression | Matches everything (always deliver) |
| Regex / wildcards | Not supported (`*`, `?`, etc. are literal characters) |

**NFC normalization** ensures that Korean, Vietnamese, and diacritic-bearing Latin scripts match correctly regardless of whether the keyword or title arrived in precomposed (NFC) or decomposed (NFD) form. For example, Korean `오류` (2 NFC runes) and its 4-rune NFD expansion are byte-different but treated as equal.

**Substring note**: `"key word"` matches a title containing `the key word here` (the phrase appears verbatim) but not `the keyword here` (the space is missing). A quoted phrase is still a *substring* match, not a whole-title equality.

## Validation and Error Handling

Expressions are validated at **write time** in the service layer:

- `Create` and `Update` compile the expression via `filter.Compile`. A parse failure returns an HTTP `400` with an `ErrValidation` service error; the message embeds the byte position, e.g. `invalid keywords expression: filter: parse error at pos 5: unexpected ','`.
- At delivery time the expression is recompiled (it is short and microsecond-cheap; no caching by design). A channel whose stored expression is somehow invalid is skipped with a log line rather than crashing the delivery loop.

`UpdateChannelParams.Keywords` is a `*string` to distinguish "field not provided" (leave unchanged) from "explicitly cleared" (set to empty → matches everything). This fixes a prior bug where clearing keywords via update was impossible.

## Rejections (Malformed Expressions)

All of the following are rejected with a `ParseError`. A stored expression can never produce an ambiguous match because invalid input never reaches storage.

| Category | Examples |
|----------|----------|
| Empty operand | `a,,b`, `a && b`, `a &`, `& a`, `,`, `a,` |
| Operator without operand | `!`, `&`, `\|`, `a \| \| b` |
| Unbalanced / empty group | `(a`, `a)`, `()`, `(a,)`, `)(a` |
| Unterminated quote | `"abc`, `"""` |
| Adjacent factors without operator | `a (b)`, `(a)(b)`, `a"b"` |
| Empty keyword | `""`, `a & ""`, `(), a` |
| Operators only | `& \| ,`, `! &`, `(!)` |

Empty keyword (`""`) is rejected because the empty string is a substring of every title and would match everything — an "unexpected surprise" source.

## Known Limitations

These are intentional, documented limitations rather than bugs:

- **German `ß`**: `strings.ToLower("Straße") == "straße"`, which does **not** equal `strasse`. A keyword `Straße` will not match a fully-uppercase title `STRASSE`. Full case folding (ß → ss) is deliberately not applied to keep behavior locale-independent and predictable.
- **Turkish `İ` / `ı`**: not locale-folded; uses Unicode default casing.
- **Emoji sequences** (ZWJ sequences like `👨‍👩‍👧`, skin-tone modifiers, flag regional indicators) are matched **byte-exactly** after NFC. Different emoji compositions (e.g. with/without ZWJ) are treated as different keywords.

## Full-width Characters

The seven operators are **ASCII-only**. Their full-width variants (`， ＆ ｜ ！ （ ）`) are **literal keyword content**, not operators — a deliberate choice that keeps the grammar unambiguous (no need to enumerate "which full-width variants count"). The trade-off is handled in the UI: Chinese/Japanese IMEs commonly emit full-width punctuation, so the frontend form shows a live parsed preview so users immediately see that `监控，告警` is a single keyword rather than an OR.

## Worked Examples

| # | Input | Meaning |
|---|-------|---------|
| 1 | `alert` | title contains `alert` |
| 2 | `a, b, c` | `a` OR `b` OR `c` |
| 3 | `a \| b \| c` | same as #2 (`\|` ≡ `,`) |
| 4 | `a & b & c` | `a` AND `b` AND `c` |
| 5 | `!test` | title does NOT contain `test` |
| 6 | `prod & (error, warning) & !debug` | `prod` AND (`error` OR `warning`) AND NOT `debug` |
| 7 | `key word 1` | keyword `key word 1` (multi-word, no quotes) |
| 8 | `! key word` | NOT keyword `key word` |
| 9 | `C++`, `a/b`, `🚀go` | each a single keyword, no quotes needed |
| 10 | `"a,b"` | keyword `a,b` |
| 11 | `"a & b"` | keyword `a & b` |
| 12 | `"say ""hi"""` | keyword `say "hi"` |
| 13 | `""""` | keyword `"` |
| 14 | `!!a` | `a` (double negation) |

## Implementation

### Backend

All logic lives in `internal/service/delivery/filter/` (~250 lines, zero new dependencies — `golang.org/x/text/unicode/norm` was already a direct dependency):

| File | Responsibility |
|------|----------------|
| `lexer.go` | Tokenizer: seven operators, bare/quoted keyword reading, `""` doubling, whitespace skipping |
| `ast.go` | AST node types: `orNode`, `andNode`, `notNode`, `keywordNode`, `alwaysTrueNode` |
| `parser.go` | Recursive-descent parser following the precedence grammar; panics into `*ParseError{Pos, Msg}` |
| `evaluator.go` | `normalizeMatch` (NFC + ToLower) and `containsSubstr` |
| `filter.go` | Public API: `Compile(expr) (*Matcher, error)`, `MustCompile(expr) *Matcher`, `(*Matcher).Match(title) bool`, `*ParseError` |

The matcher is invoked from `internal/service/delivery/post_delivery.go`. The check is hoisted **above** the `switch channel.Kind`, so all channel kinds share the same filter (previously it was wired only into the Feishu branch).

### Frontend

`src/lib/keyword-filter.ts` is a TypeScript port of the grammar used **only for live form validation and preview** — the backend remains authoritative. It exposes `compileKeywordFilter(expr)` (returns `{ node, error }`) and `describeFilter(node)` (renders a human-readable description with precedence parentheses, e.g. `a | (b & c)`). `src/components/settings/DeliveryChannelForm.tsx` renders a `KeywordFilterFeedback` line under the input: a readable preview when valid, a red error message when not.

### Performance

Benchmarked in `internal/service/delivery/filter/filter_bench_test.go` (AMD Ryzen 5 5600H):

| Scenario | ns/op | allocs/op |
|----------|------:|----------:|
| Compile, single keyword | ~153 | 4 |
| Compile, compound (`prod & (error,warning,fatal) & !debug & !(staging,local)`) | ~1152 | 24 |
| Match (medium title, hit) | ~682 | **0** |
| Match (long 4KB title, hit) | ~9658 | **0** |
| Normalize (NFC+ToLower), long title | ~5549 | 0 |
| Compile + Match (medium title) — real per-channel delivery cost | ~1618 | 17 |

Evaluation is **zero-allocation**; only compilation allocates (proportional to AST size). For a typical channel the per-delivery cost is ~1.6 µs. The dominant factor for long titles is title normalization (NFC + ToLower), not parsing.

## Test Coverage

Tests live in `internal/service/delivery/filter/`:

| File | Scope |
|------|-------|
| `filter_test.go` | Semantics, precedence, special characters/quoting, empty-matches-all, valid edge cases, invalid rejections |
| `filter_multilingual_test.go` | Chinese (no word boundaries), Japanese (mixed scripts, long vowel mark, small kana), Korean (NFC/NFD), Thai, Arabic/Hebrew (RTL), Cyrillic (case), German (umlauts, ß limit), Latin diacritics, emoji (ZWJ/skin tone/flags), mixed scripts, full-width literalization |
| `fuzz_test.go` | `FuzzCompile_NeverPanics` (1.3M+ execs, no panic) plus five boolean-identity property tests: De Morgan, double negation, commutativity, distributivity, tautology/contradiction |

## Design Decisions

Recorded so the rationale survives alongside the spec.

1. **Standard boolean algebra over Ansible-style flat reordering.** Ansible reorders `&`/`!` terms rather than building an AST, which cannot express `A & (B | C)`. Standard precedence + parentheses is the only design that is simultaneously complete, unambiguous, and matches most users' existing intuition.
2. **Spaces as content (Model 2).** Chosen over "spaces as separators" (search-engine style) because it matches the motivating examples verbatim and avoids frequent quoting of multi-word phrases. The cost — adjacent factors need explicit operators — is exactly what guarantees no ambiguity.
3. **`""` doubling over `\"` escaping.** Doubling introduces no escape character, so `\` is unconditionally literal (keywords like `C:\` and `a\b` work everywhere). CSV/SQL precedent.
4. **Comma now means OR (breaking change).** The pre-v0.1.0 semantics treated comma as AND; storing `a, b` and shipping to a new-OR evaluator would silently loosen filtering. Accepted as a documented breaking change at v0.1.0 (channels feature was new, low blast radius). The DB column is unchanged — only its interpretation changed.
5. **NFC normalization.** Without it, Korean/Vietnamese/accented-Latin users would see silent cross-platform mismatches (keyword in NFC, title in NFD). Required `golang.org/x/text`, already a dependency.
6. **Unicode default case folding (not full case folding).** Keeps behavior locale-independent and predictable; `ß↔ss` and Turkish `İ/ı` are documented limitations.
7. **ASCII-only operators.** Full-width variants are literal content, keeping the grammar unambiguous. The frontend live preview absorbs the UX cost for CJK IME users.
