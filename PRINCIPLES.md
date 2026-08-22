# Coding principles

Superseded by [AGENTS.md § Conventions](AGENTS.md#conventions) as the live home for rules; kept as an archive until migration completes, no further updates.

Behavioral constraints for the agent. Each is a rule the agent gets wrong without being told. Production safety and data integrity outrank every principle here — including the license to redesign from scratch; against a mere default or style rule, the principle wins.

## Ground every conclusion in fact

Library facts, APIs, and protocols must be read from source or docs before you act on them — training data is a blind spot, not a source. Verify every conclusion on the ground: `file:line` for logic, Playwright for UI, `docker exec markpost-postgres psql` for data, read-only HTTP against dev for behavior — means the model can actually perform. Pure algorithm or syntax knowledge may use training knowledge.

The `docker compose run` flag incident is the shape of getting this wrong: ansible's `interactive=false` would emit `--no-interactive`, a flag docker compose does not have — caught only because it was checked against the compose docs (v5.1.3) before relying on it.

## Defer to community convention

When a convention or best practice is uncertain, ask "what is the community/official convention?" and verify against authoritative open-source source, not training memory (e.g. whether `fmt`/`lint` are the prek group names here, or how golangci-lint v2 wants its config — both verifiable against the tool's own schema).

Distinct from _Ground every conclusion in fact_: that one governs facts about a library you are integrating; this one governs convention and best-practice decisions.

## Converge before you implement

A spec or plan must be self-contained, complete, and unambiguous — an executor with no taste can land it mechanically, with no room to improvise. Resolve every open point before implementing; do not start on the strength of a half-settled plan.

## Fix the root cause, not the symptom

The solution you choose must be the most natural and optimal — not a patch over the symptom, and not one trapped by the existing implementation. You may shed all legacy and start from zero when the root fix requires it.

When formatter/lint coverage leaked across tools, the fix was not to configure each AI hook separately but to delegate to `prek.toml` as the single truth — no parallel formatter or lint definition exists outside prek.

## Design from first principles

Derive a design from the business essence; every premise is breakable; an elegant scheme beats an inherited one. Distinct from _Fix the root cause, not the symptom_: that one is how you _fix_ a problem (root, not patch); this one is how you _design_ a system (re-derive, question assumptions). Dropping sqlite/mysql entirely instead of carrying three drivers forever is this principle applied.

## Single source of truth

Each category of information has exactly one authoritative source: schema truth is the embedded golang-migrate SQL files (paired with every GORM tag change), API truth is the generated Swagger in `backend/docs/` (regenerated, never hand-edited), UI text lives in the locale files, deploy config in ansible group_vars + templates. The frontend renders; it does not decide.

## Naming is part of the API

A name is an API surface. If a name does not fit its business meaning, do not force it — brainstorm candidates and let the user choose, to prevent semantic drift.

## Degrade gracefully, never silently

A failure must be handled and observably recorded, and must not block downstream work — but a silent failure is always wrong. A failed delivery (webhook, Feishu) must not block the post, but it must land in delivery history, not vanish; config validation fails loudly at startup instead of falling back to defaults no one knows about.

## Minimal mock, maximal real

Mock only the request boundary, never the whole service. Backend tests run against a real PostgreSQL container via testcontainers-go — repository mocks in CI hide SQL drift; e2e drives the real container image through the public URL; local and CI run the same suite as fully as feasible.
