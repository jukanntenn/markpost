# MRFC: Worktree development-environment isolation

Status: proposed

English | [中文](2026-08-22-worktree-dev-environment-isolation.zh.md)

## Problem

The [development loop](../implemented/2026-08-22-agent-driven-development-loop.md) places one git worktree per stack layer under `.local/worktrees/`, but the compose dev environment names its containers unconditionally — `markpost-backend`, `markpost-frontend`, `markpost-postgres` — and binds one shared `markpost_pgdata` volume. Two consequences: only one dev environment can run at a time across the main checkout and every worktree (starting a worktree's environment requires stopping the main checkout's), and v1 of the loop accepted that mutual exclusion as a known limit, serializing any verification that needs a running stack. Unit tests and testcontainers-backed backend tests are unaffected — they need no compose environment — so the constraint bites exactly where full-stack or frontend-against-backend verification is wanted in parallel.

## Proposal

Parameterize the dev environment by an environment name. `devops/dev.py` accepts `--env <name>` (defaulting to the unprefixed `markpost`), which sets the compose project name and prefixes every container name and the postgres data volume — `wt-123-markpost-backend`, `wt-123_markpost_pgdata` — through the environment variables the compose file already interpolates. The main checkout keeps today's unprefixed names exactly, so existing muscle memory, the AGENTS.md `docker exec markpost-postgres` instruction, and any scripts addressing containers by name continue to work; a worktree session passes `--env wt-<issue>` and gets a fully disjoint environment. Host port bindings must also parameterize (an offset or explicit mapping per environment) or the second environment's port claims collide with the first's; the compose file gains the interpolation and `dev.py` the flag plus a collision check that fails loud with the occupying environment named. The `dev-loop` skill passes the issue-derived environment name automatically when it drives verification inside a worktree.

## Alternatives considered

**Keep v1's serial usage.** No work, and the loop functions — but every full-stack verification serializes against every other, and the scheduled driver (phase 2) multiplies the contention exactly when nobody is watching to stop the main environment.

**Per-worktree compose override files.** A generated `docker-compose.override.yml` per worktree avoids touching the main file, but generated drift between the base and its overrides becomes its own failure mode, and the override must still invent names — the same parameterization by another, less visible route.

**One shared environment, databases separated by name.** Cheapest isolation of data only; the backend and frontend containers themselves remain singletons, so code changes from two worktrees still cannot coexist in one running environment.

## Acceptance criteria

Two dev environments — the main checkout's unprefixed one and a `--env wt-<n>` worktree one — run concurrently without name, volume, or port collisions, each serving its own checkout's code; the main checkout's container names, volume names, and host ports are byte-identical to today's; `dev.py start` without `--env` behaves exactly as before; a requested environment whose ports are occupied fails with the occupying environment named.

## Risks

Compose-file interpolation adds indirection to a file operators read as configuration; keeping every default identical to today's unprefixed names bounds that cost. Port parameterization risks drift between the backend and frontend port wiring if the two disagree — the collision check exists to fail that loudly rather than mysteriously. Each concurrent environment consumes its full container and memory footprint; parallelism is opt-in per `--env`, never the default. And postgres major-version upgrades across simultaneously running environments share one image, so the volume-per-environment split must be respected by any future migration tooling.
