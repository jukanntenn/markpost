# MRFC: Keyword filter input UX — live natural-language feedback

Status: implemented

English | [中文](2026-08-31-keyword-filter-input-ux.zh.md)

## Problem

The keyword filter field asked users to learn a seven-operator boolean grammar from one compressed one-liner — `语法：逗号/竖线=或，&=且，!=非，""=精确短语，()=分组` — which was also wrong: `""=精确短语` misdescribes quoting (quotes make operator characters literal; matching stays substring, and `""` is the doubled literal quote). Parse failures surfaced raw internal English errors (`unexpected comma`, `expected ')', got eof`). The `[?]` help tooltip never appeared on hover at all: its `group-hover:block` had no `group` ancestor, so only the click toggle worked. Meanwhile the spec promised better: [the keyword filter spec](../../../specs/backend/keyword-filter.md) designates a live parsed preview as the mitigation for CJK IME users whose full-width `，` is literal content, and its § Frontend documented `describeFilter` plus a feedback component — but the July dialog rewrite (D5.6) dropped semantic interpretation, keeping syntax validation only, without updating the spec. Users could not see what an expression does, and the spec–code contract was broken.

## Decision

The keyword field in `src/components/delivery/DeliveryChannelDialog.tsx` is a three-layer feedback loop, backed by `src/lib/keyword-filter.ts`:

**A live natural-language preview (cron-guru style).** `describeFilter(node, phrasebook)` renders the AST as a locale sentence — zh-Hans: `标题包含「prod」且（包含「error」或包含「warning」）且不包含「debug」时推送`; en: `Delivers when the title contains “prod” and (contains “error” or contains “warning”) and does not contain “debug”`. The walker owns structure — precedence-aware parentheses (OR wrapped inside AND and vice versa), double-negation folding, 30-code-point keyword truncation — while each locale supplies connectives through the `Phrasebook` interface; sentences are composed from the `keywordsPreviewSentence` / `keywordsPreviewAlways` messages in all four locales (en, zh-Hans, zh-Hant, ja). The empty expression previews as "delivers every post". This finally delivers the spec's full-width mitigation: `监控，告警` visibly previews as one keyword.

**Structured localized errors.** `FilterParseError` carries `{ code, pos, token }` — one of four codes (`unterminated_quote`, `unexpected_token`, `missing_rparen`, `empty_keyword`), a 1-based code-point position (astral characters count as one; null when input ran out), and the offending token kind. The dialog maps these to locale messages such as `语法错误：第 3 个字符附近不应出现「,」，这里需要一个关键词或「(」`; internal token names never reach the user.

**A syntax help popover with click-to-fill examples.** The `[?]` trigger opens a `src/components/ui/popover.tsx` panel (base-ui Popover, keyboard-focusable, `z-[100]` per the dialog-popup convention) holding the operator cheat sheet, the three gotchas (spaces are content, full-width punctuation is literal, empty matches all), and five example chips. Picking a chip fills the input and closes the popover so the preview is immediately visible. The broken hover span is gone.

The spec's § Frontend is rewritten to describe this reality, ending the drift the D5.6 rewrite introduced.

## Alternatives considered

**Keep validation-only (the D5.6 posture).** Cheapest, but the spec had already promised the live preview as the CJK full-width mitigation — without it `监控，告警` silently matches nothing the user intended, and the drift stays.

**Condition-list rendering** (top-level AND/OR decomposed into bullet conditions). Reads best for compound AND chains, but is multi-line inside a `sm:max-w-lg` dialog and over-signals the simple `mark, post` case. The single-sentence form was chosen — cron-guru parity, and simple expressions degrade to a natural short sentence.

**Revive the old `describeFilter` (re-rendered normalized expression, `a | (b & c)`).** Cheapest to port, but re-rendering the expression in the same notation explains nothing — the ask was natural language, not pretty-printing.

**Localize errors server-side.** Backend 400 messages would need locale awareness in the API layer; the form validates client-side with the same grammar before any request, so the client is where users meet errors. Direct API users keep the English message with byte position.

**A tooltip instead of a popover for help.** One hover line cannot hold a cheat sheet plus examples, and hover-only is unusable on touch; the popover is click/keyboard driven.

## Consequences

Users can now write expressions by watching the preview narrate the effect, recover from errors at a pointed character position in their own language, and learn the grammar from the cheat sheet by trying examples in place. The costs: four locale files each carry ~28 new message keys; a future fifth locale must implement the `Phrasebook` connectives; and the TS walker's rendering must stay semantically aligned with the Go grammar — an extension of the port contract `keyword-filter.ts` already carries (backend authoritative, frontend mirrors). Verification shipped with the change: per-locale walker snapshots (en, zh-Hans) and structured-error positions in `keyword-filter.test.ts`, dialog component tests covering preview/error/help/chip, e2e preview assertions in `e2e/tests/delivery-channel.spec.ts`, and interactive verification of the zh-Hans flow (compound preview, full-width single-keyword preview, localized error, popover fill-and-close).
