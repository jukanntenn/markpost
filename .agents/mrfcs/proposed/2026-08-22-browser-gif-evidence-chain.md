# MRFC: Browser GIF evidence for UI changes

Status: proposed

English | [中文](2026-08-22-browser-gif-evidence-chain.zh.md)

## Problem

The [development loop](2026-08-22-agent-driven-development-loop.md) accepts Playwright screenshots as the evidence standard for UI-changing layers: they prove that a state renders, and nothing more. Interaction is exactly what screenshots cannot carry — transitions, animation timing, hover and focus behavior, multi-step flows — and it is also where UI regressions hide. The reference harness records a short browser GIF for GUI-changing pull requests and treats the recording as part of the evidence chain; markpost's v1 deferred that machinery deliberately, so this record holds the upgrade path and the trigger for taking it.

## Proposal

Every implementation layer that changes user-visible frontend behavior attaches, alongside the existing screenshots, one short screen recording (a few seconds, GIF or silent MP4) exercising the changed interaction against the dev environment: the action performed, the transition observed, the state settled. Recording runs through the already-required Playwright verification session (`playwright-cli` video capture or `ffmpeg` assembled from screenshots), scripted by the `dev-loop` verification step so the cost is one flag, not a manual chore. Evidence lands in the pull-request body next to the screenshots; recordings are never committed to the repository. The loop's acceptance mapping gains one line per UI layer: the interaction demonstrated, the recording linked. Non-UI layers are unaffected.

## Alternatives considered

**Keep screenshots only.** Zero added cost, but interaction claims stay unverified prose until a human clicks through by hand — precisely the review the loop removes from the human's day.

**Full video with narration.** Richer evidence, disproportionate production cost per pull request, and unreviewable at a glance; a short silent clip carries the delta.

**Interactive review environments (ephemeral preview deployments per pull request).** The strongest evidence and the heaviest machinery — preview infrastructure for a static-export frontend served from a single image; far beyond what interaction verification needs.

## Acceptance criteria

A UI-changing implementation layer cannot pass the agent's pre-review without an embedded recording demonstrating the changed interaction; the recording step is a scripted part of verification, not a judgment call; a layer with no user-visible interaction change is explicitly marked as such in the evidence section.

## Risks

Recordings bloat pull-request bodies if unbounded — the few-seconds rule and a size cap in the skill keep them small, with hosting fallback (issue comment attachment) if body embedding degrades. Capture flakiness on animation timing adds a verification-retry cost. The trigger to build this at all is repeated human re-verification of interaction claims during review; until that signal appears, screenshots remain the accepted standard and this proposal waits.
