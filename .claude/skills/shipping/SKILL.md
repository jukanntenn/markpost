---
name: shipping
description: Ship the current work end-to-end — quality gate, commit, build & push the image, deploy, report. Use when the user asks to ship or deploy the current changes to a server (dev/staging/production). For a versioned public release (version bump, changelog, tag, Docker Hub), use the release skill instead.
disable-model-invocation: true
---

Ship the current work in one pass: gate → commit → build & push → deploy → report.

1. **Gate.** prek owns every gate — run it from the repo root: `prek run --all-files` (fmt + lint + generated-files drift; exactly CI's Lint job), then `prek run --stage pre-push --all-files` (backend `go build` + `go test -race`, frontend `pnpm test:run` + `pnpm build`; needs Docker — testcontainers). Any failure: stop, fix, re-run until green.
2. **Commit.** Delegate to the commit skill.
3. **Build & push.** `python3 docker/build.py --push` — rolling `main` tag, default registry. Add `--all-platforms` or `--tags <t>` only if the user asks.
4. **Deploy.** `ansible-playbook devops/ansible/deploy.yml` from the repo root — `dev` (fn) is the default target; `-e target=staging|production` only if the user names one. The playbook orders postgres → `migrate up` → app and its trailing health + version check owns verification — do not re-verify by hand.
5. **Report.** Image reference, environment, outcome — one line.
