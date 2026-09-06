# MRFC: Site footer for the landing page and the app shell

Status: implemented

English | [中文](2026-09-05-site-footer.zh.md)

## Problem

Only the landing page had a footer — the colophon: a closing CTA, one centered row of three links, a copyright/typeset line. The app shell (dashboard and every admin page) ended at the last card with nothing below. The running version appeared nowhere in the UI: a self-hoster cannot tell from the console which release an instance runs, and bug reports arrive without it. The landing footer also carried wayfinding for only three of the project's real destinations.

## Decision

A two-surface footer set, benchmarked on the GitHub/Vercel footer skeleton (brand block + link columns + bottom bar) scaled down to the links that actually exist:

- **Landing (`SiteFooter.tsx`, replacing `ColophonFooter`)**: the closing CTA is retained at the top; below it a brand block (logo + one-line tagline) beside two link columns — *Resources* (docs, issue feedback) and *Project* (GitHub, Docker Hub, MIT License) — and a bottom bar: `© {year} markpost · MIT License · v{APP_VERSION}` with the typeset line kept on the right. Columns stack on mobile.
- **App shell (`AppShell.tsx`)**: a slim footer pinned to the page bottom — `markpost v{APP_VERSION}` left, docs + GitHub links right, aligned to the same 1200px container as the top bar. Auth pages keep no footer (the login card already links home).
- **Version source**: `APP_VERSION` in `src/lib/site.ts` imports `version` from `package.json` — the file the release flow bumps; the only source, never hardcoded elsewhere. Site-wide link constants (`REPO_URL`, `DOCS_URL`, `ISSUES_URL`, `LICENSE_URL`, `DOCKER_HUB_URL`) moved there from `components/landing/links.ts`, which keeps only `LANDING_CONTAINER`.
- Copy: `landing.colophon` gains `tagline`/`resources`/`project`/`issues`; a new `footer` namespace (dashboard link labels) — both across all four locales.

No invented destinations: status page, support, and socials do not exist, so they get no footer links.

## Alternatives considered

**Keep the colophon as the footer.** The colophon was a deliberate book-design signature, but the maintainer's explicit direction was a professional footer benchmarked on well-known services; incumbency is not authority. The typeset line survives in the bottom bar so the signature is not lost, only demoted.

**Single-row link footer (β option).** Lighter, but a brand block plus columns is the structure that reads as "professional footer"; with only five real links the structure comes from layout, not from padding columns.

**Version from the backend `/api/v1/version` endpoint.** The endpoint reports the binary's git-describe string; the frontend is a static export whose identity is fixed at build time, so the bundled `package.json` version is the build's own truth, available offline and without a fetch race on every shell render.

**Footer on the auth pages too.** A footer under a focused login card adds noise; the card already carries the markpost home link.

## Consequences

The landing carries the full footer; the console shows its running version — the piece self-hosted operators actually need. `specs/frontend/routes.md` (+ zh twin) §06 is rewritten in the same change. Two costs: the footer link set must be pruned if a destination dies (they all live in one file now, `src/lib/site.ts`), and the landing's `landing.colophon` namespace now mixes CTA and footer copy — accepted because they render in one component; the CTA keys (`heading`/`subheading`/`getStarted`) and footer keys (columns, bottom bar) remain distinct within it.
