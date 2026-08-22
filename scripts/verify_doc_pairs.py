#!/usr/bin/env python3
# Bilingual documentation pairs (docs/AGENTS.md rule 7): every
# documentation file ships as foo.md (English) beside foo.zh.md (Chinese),
# equal authority, the pair updating together. Checks: pair completeness
# in both directions, a language switcher linking the twin in the header
# region, link locale (corpus targets use the reader's own locale),
# structural parity (identical heading-depth sequences and byte-identical
# fenced code blocks, comments included — examples are not translated),
# and language purity (no CJK prose in .md; .zh.md legitimately carries
# English terms). Exemptions and CJK allowances live in
# scripts/doc_languages.manifest.json. With file arguments the scope
# restricts to those files expanded to whole pairs, and a not-yet-written
# .zh.md link target is tolerated mid-corpus; the strict full-corpus run
# is CI's. Part of doc_sync.py.
from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from urllib.parse import unquote

from doclib import ROOT, collect, extract_links, is_external, mask_source, restrict

PATTERNS = [
    "README.md",
    "README.zh.md",
    "PRINCIPLES.md",
    "PRINCIPLES.zh.md",
    "docs/*.md",
    "specs/*.md",
    "specs/**/*.md",
    ".agents/mrfcs/*.md",
    ".agents/mrfcs/**/*.md",
    "e2e/*.md",
]

MANIFEST = ROOT / "scripts" / "doc_languages.manifest.json"
MANIFEST_SECTIONS = {"excluded", "cjk_allowed"}
CJK = re.compile(r"[\u3000-\u303f\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff\uff00-\uffef]")
HEADING = re.compile(r"^(#{1,6})\s")
SWITCHER_REGION = 10


def is_zh(path: Path) -> bool:
    return path.name.endswith(".zh.md")


def zh_twin(path: Path) -> Path:
    return path.with_name(path.name[: -len(".md")] + ".zh.md")


def en_twin(path: Path) -> Path:
    return path.with_name(path.name[: -len(".zh.md")] + ".md")


def path_part(url: str) -> str:
    return unquote(url.split("#", 1)[0].split("?", 1)[0])


def load_manifest(problems: list[str]) -> tuple[set[str], set[str]]:
    data = json.loads(MANIFEST.read_text(encoding="utf-8"))
    unknown = sorted(set(data) - MANIFEST_SECTIONS)
    if unknown:
        problems.append(f"{MANIFEST.name}: unknown section(s) {unknown}")
    excluded: dict[str, str] = data.get("excluded", {})
    cjk_allowed: dict[str, str] = data.get("cjk_allowed", {})
    for rel in sorted(set(excluded) | set(cjk_allowed)):
        if not (ROOT / rel).exists():
            problems.append(f"{MANIFEST.name}: {rel} does not exist — update the manifest in the same change")
    for rel in sorted(cjk_allowed):
        if rel not in excluded and not rel.endswith(".md"):
            problems.append(f"{MANIFEST.name}: cjk_allowed entries name .md files whose prose needs CJK")
    for rel in sorted(excluded):
        if rel.endswith(".md") and not rel.endswith(".zh.md") and zh_twin(ROOT / rel).is_file():
            problems.append(f"{rel}: excluded from pairing — its {zh_twin(Path(rel)).name} twin must go")
    return set(excluded), set(cjk_allowed)


def excluded_match(rel: str, excluded: set[str]) -> bool:
    return any(rel == e or (e.endswith("/") and rel.startswith(e)) for e in excluded)


def heading_levels(masked: str) -> list[int]:
    return [len(m.group(1)) for line in masked.split("\n") if (m := HEADING.match(line))]


def fenced_blocks(text: str) -> list[str]:
    blocks: list[str] = []
    current: list[str] | None = None
    for line in text.split("\n"):
        if current is None:
            if line.lstrip().startswith("```"):
                current = [line]
        else:
            current.append(line)
            if line.lstrip().startswith("```"):
                blocks.append("\n".join(current))
                current = None
    return blocks


def has_switcher(masked: str, twin: Path) -> bool:
    return any(
        link.line <= SWITCHER_REGION and path_part(link.url) == twin.name
        for link in extract_links(masked)
    )


def locale_violations(path: Path, masked: str, corpus: set[str], restricted: bool) -> list[str]:
    rel = path.relative_to(ROOT).as_posix()
    want_zh = is_zh(path)
    own_twin = (en_twin(path) if want_zh else zh_twin(path)).name
    out: list[str] = []
    for link in extract_links(masked):
        if link.line <= SWITCHER_REGION and path_part(link.url) == own_twin:
            continue  # the language switcher is the one sanctioned cross-locale link
        target = path_part(link.url)
        if not target or is_external(link.url):
            continue
        resolved = (path.parent / target).resolve()
        if not resolved.is_file():
            continue
        try:
            trel = resolved.relative_to(ROOT).as_posix()
        except ValueError:
            continue
        if trel not in corpus:
            continue
        if want_zh and not trel.endswith(".zh.md"):
            required = trel[: -len(".md")] + ".zh.md"
            if restricted and not (ROOT / required).is_file():
                continue  # counterpart not translated yet — the strict full run owns the end state
            out.append(f"{rel}:{link.line}: link to {target} must use the .zh.md variant on a Chinese page")
        elif not want_zh and trel.endswith(".zh.md"):
            out.append(f"{rel}:{link.line}: link to {target} must use the .md variant on an English page")
    return out


def purity_violations(path: Path, masked: str) -> list[str]:
    rel = path.relative_to(ROOT).as_posix()
    twin_name = zh_twin(path).name
    out: list[str] = []
    for i, line in enumerate(masked.split("\n"), 1):
        if f"]({twin_name}" in line:
            continue  # the switcher names the twin in Chinese by convention
        m = CJK.search(line)
        if m:
            out.append(f"{rel}:{i}: CJK character {m.group(0)!r} in English prose — that content belongs in the .zh.md twin")
    return out


def pair_violations(en: Path, zh: Path, corpus: set[str], cjk_allowed: set[str], restricted: bool) -> list[str]:
    en_rel = en.relative_to(ROOT).as_posix()
    zh_name = zh.name
    en_text = en.read_text(encoding="utf-8")
    zh_text = zh.read_text(encoding="utf-8")
    en_masked = mask_source(en_text)
    zh_masked = mask_source(zh_text)
    out: list[str] = []
    if not has_switcher(en_masked, zh):
        out.append(f"{en_rel}: no language switcher linking {zh_name} in the first {SWITCHER_REGION} lines")
    if not has_switcher(zh_masked, en):
        out.append(f"{zh.relative_to(ROOT)}: no language switcher linking {en.name} in the first {SWITCHER_REGION} lines")
    if heading_levels(en_masked) != heading_levels(zh_masked):
        out.append(f"{en_rel}: heading-depth sequence diverges from {zh_name}")
    if fenced_blocks(en_text) != fenced_blocks(zh_text):
        out.append(f"{en_rel}: fenced code blocks differ from {zh_name} — code and examples stay byte-identical, comments included")
    out += locale_violations(en, en_masked, corpus, restricted)
    out += locale_violations(zh, zh_masked, corpus, restricted)
    if en_rel not in cjk_allowed:
        out += purity_violations(en, en_masked)
    return out


def main(argv: list[str]) -> int:
    problems: list[str] = []
    excluded, cjk_allowed = load_manifest(problems)
    scope = [p for p in collect(PATTERNS) if not excluded_match(p.relative_to(ROOT).as_posix(), excluded)]
    corpus = {p.relative_to(ROOT).as_posix() for p in scope}

    restricted = bool(argv[1:])
    if restricted:
        pairs: set[Path] = set()
        for p in restrict(scope, argv[1:]):
            pairs.add(p)
            twin = en_twin(p) if is_zh(p) else zh_twin(p)
            if twin.is_file():
                pairs.add(twin)
    else:
        pairs = set(scope)

    verified = 0
    for en in sorted(p for p in pairs if not is_zh(p)):
        rel = en.relative_to(ROOT).as_posix()
        zh = zh_twin(en)
        if not zh.is_file():
            problems.append(f"{rel}: in-scope documentation ships as a pair — write its .zh.md twin")
            continue
        problems += pair_violations(en, zh, corpus, cjk_allowed, restricted)
        verified += 1
    for zh in sorted(p for p in pairs if is_zh(p)):
        if not en_twin(zh).is_file():
            problems.append(f"{zh.relative_to(ROOT)}: .zh.md without its .md original — an orphan twin cannot pair")

    if problems:
        print(f"verify_doc_pairs: bilingual pairing rules violated (see docs/AGENTS.md rule 7):", file=sys.stderr)
        print("\n".join(f"  {p}" for p in problems), file=sys.stderr)
        return 1
    print(f"verify_doc_pairs: {verified} pair(s) verified (completeness, switchers, link locale, structure, purity)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
