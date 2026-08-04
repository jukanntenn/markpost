# Known Issues

## Markdown: home-directory `~` rendered as strikethrough

**Status:** Accepted, not fixed (2026-06-25).

### Symptom

A literal `~` in post prose (most often a home-directory path such as
`~/GitHub` or `~/.claude/skills`) is interpreted as a strikethrough delimiter.
When a paragraph contains two such `~` characters, everything between them is
wrapped in `<del>` and both tildes are consumed, producing visibly struck-through
text and missing tildes.

Reproduced on production post `p-VifrCx3z7poWITSWe7VQQ`: the paragraph about
软链接管理 AI Agent Skills renders as

> ...存放在 <del>/GitHub 目录，再用 .claude/skills → .agents/skills ...
> （/.agents/skills、/.codex/skills、</del>.claude/skills），导致管理混乱。

Note the swallowed `~` in both `~/GitHub` and `~/.claude/skills`.

### Root cause

The renderer (`backend/internal/service/post/post.go`) builds goldmark with
`extension.GFM`. goldmark's strikethrough extension treats a **single** `~` as a
valid strikethrough delimiter
(`contexts/goldmark/extension/strikethrough.go`: `ScanDelimiter(line, before, 1, ...)`,
min length 1; test case `~Hi~` -> `<del>Hi</del>`).

This is **GFM-spec-compliant**: GFM §6.5 says "one or two tildes" and Example 491
renders `~there~` as `<del>there</del>`. goldmark merely follows the spec, and it
exposes no option to require `~~`. The flanking rules then legitimately pair two
single tildes that appear in the same paragraph — e.g. an opening `~/GitHub`
(preceded by whitespace) with a closing `~/.claude/skills` (preceded by the CJK
punctuation `、`), which is why this post is affected.

The wider ecosystem considers single-tilde strikethrough a defect:
cmark-gfm (GitHub's own GFM reference) merged PR #362 in 2025-03 to require `~~`,
and markdown-it / GitLab / VS Code only support `~~`.

### Decision

Left as-is for now. Strikethrough via `~~` continues to work; only single-`~`
content is at risk.

### Recommended fix (when revisited)

Replace `extension.GFM` with its four sub-extensions, substituting a custom
double-tilde strikethrough extender (reuse `parser.ScanDelimiter(..., 2, ...)`,
`extension/ast.NewStrikethrough()`, `extension.NewStrikethroughHTMLRenderer()`).
`~~x~~` still strikes; `~x~` and `~/path` render literally; `~~~` is still
rejected (GFM Example 493). No data migration is needed because posts are
rendered on read.

### References

- goldmark strikethrough: `contexts/goldmark/extension/strikethrough.go`
- GFM spec §6.5: https://github.github.com/gfm/#strikethrough-extension-
- cmark-gfm require-double-tilde: https://github.com/github/cmark-gfm/pull/362
- cmark-gfm #99 (single tilde is problematic): https://github.com/github/cmark-gfm/issues/99

## Config file name: `markpost.toml` backward-compat fallback pending removal

**Status:** Resolved (2026-07-27). The `markpost` fallback name and its test were removed; staging mount retargeted to `/app/config.toml`.

### Background

In the single Docker container the Go binary is named `markpost` and runs in
`/app`, so the old auto-discovered config file `/app/markpost.toml` collided
with the binary name. The default config file was renamed from `markpost.toml`
to `config.toml`. To avoid breaking existing deployments that still mount
`/app/markpost.toml`, the config loader (`config.go`) keeps a temporary
fallback: it searches for `config` first, then `markpost`.

### Follow-up work

Once all environments (dev / staging / production) have switched their volume
mounts to `:/app/config.toml:ro` and run an image with the new loader:

1. Remove the `"markpost"` candidate from the name list in
   `backend/internal/config/config.go` (`loadConfig` else-branch loop).
2. Delete the `TestLoad_AutoDiscoveryMarkpostTomlFallback` test in
   `backend/internal/config/config_test.go`.
3. Run `go test ./... && golangci-lint run` and commit as a single cleanup.

### Verification of migration completion

Confirm no deployment still references the old mount target before removing the
fallback:

```bash
# Ansible templates (all three should be /app/config.toml)
rg -n 'config_file' devops/ansible/templates/*/docker-compose.yml.j2

# Any stray mount references
rg -rn '/app/markpost.toml' .
```

### References

- Config loader: `backend/internal/config/config.go`
- Fallback test: `backend/internal/config/config_test.go`
- Deployment mounts: `docs/deployment.md`

## Markdown: CJK fullwidth punctuation breaks emphasis closing

**Status:** Resolved (2026-07-27).

### Symptom

Emphasis delimiters (`**`) adjacent to CJK fullwidth punctuation (e.g. the
fullwidth right parenthesis `）` U+FF09, fullwidth comma `，`, fullwidth period
`。`) fail to close, causing mis-paired `**` sequences in mixed CJK text.

Reproduced on production post `p-2lsoUuYpibCUAMN96PrvC`: the source
`本项目对**有限...的**无条件...定理）**进行...` rendered with the first `**`
literal and the wrong span bolded, because the `**` after `）` could not close.

### Root cause

CommonMark's emphasis flanking rules (spec 6.2) classify Unicode punctuation
specially: a `**` immediately preceded by punctuation cannot close emphasis.
goldmark's `util.IsPunctRune` (`unicode.IsSymbol || unicode.IsPunct`) classifies
CJK fullwidth punctuation as punctuation, so `定理）**` fails the close check
and the `**` sequences pair incorrectly.

This is a known CommonMark deficiency for CJK languages. The goldmark author
acknowledged it in issue #61: "CommonMark spec are created by westerners, so
they did not take account into our languages."

### Fix

Register a custom emphasis `InlineParser` (`internal/service/post/cjk_emphasis.go`)
that scans delimiters with a CJK-aware punctuation predicate, treating CJK
fullwidth punctuation as neutral (neither punctuation nor whitespace) so `**`
closes normally. Uses only exported goldmark APIs (InlineParser interface +
ScanDelimiter); no fork. The default emphasis parser is shadowed by registering
the custom one at priority 600 (> default 500).

### References

- goldmark emphasis flanking: `parser/delimiter.go:113 ScanDelimiter`
- goldmark IsPunctRune: `util/util.go:831`
- goldmark issue #61 (CJK emphasis): https://github.com/yuin/goldmark/issues/61
- CommonMark spec 6.2: https://spec.commonmark.org/0.30/#emphasis-and-strong-emphasis
