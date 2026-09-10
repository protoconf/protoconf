---
phase: quick-260901-wom
plan: 01
subsystem: infra
tags: [ci, trunk, buf, renovate, github-actions, gotestsum]

requires:
  - phase: quick-260901-vaj
    provides: Go floor raised to 1.25.8, 5 security dependency bumps merged (the structural cause this plan's renovate fix addresses)
provides:
  - Lint workflow (trunk hold-the-line check + buf breaking) gating every PR to main
  - Repaired trunk.yaml (buf-lint disabled, golangci-lint removed, Go runtime pin raised to 1.25.8)
  - Repaired buf.yaml (build.excludes covering the bad_proto test fixture)
  - Hardened go.yml (contents:read, concurrency group, job timeout, pinned test reporter)
  - Consolidated renovate.json (single config, grouped gomod PRs, non-automerging vulnerability alerts)
affects: [ci, dependency-management, future golangci-lint re-enablement]

actuals:
  tokens: 1100
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns: ["trunk hold-the-line lint on pull_request only (never push)", "buf breaking with --against-config to avoid a stale main-side config", "renovate packageRules grouping to prevent mutually-conflicting dependency PRs"]

key-files:
  created:
    - .github/workflows/lint.yml
  modified:
    - .trunk/trunk.yaml
    - buf.yaml
    - .github/workflows/go.yml
    - renovate.json
    - .github/renovate.json (deleted)

key-decisions:
  - "golangci-lint removed entirely from trunk.yaml (not re-pinned) — trunk CLI install failed (needs sudo, no password available in this sandbox), and no v1-line pin can typecheck a go1.25.8 module. go build/vet/test already cover Go correctness; a human with local trunk can re-resolve the pin later."
  - "buf-lint moved to trunk's lint.disabled (not deleted) — 73 existing findings require enum/package renames that break wire compatibility, a hard CLAUDE.md constraint. Left visible as future work rather than silently dropped."
  - "gotestsum pinned to v1.13.0 and the pipe replaced with its --raw-command replay form — verified locally against a real go test -json output before committing, per the plan's explicit requirement."

requirements-completed: [QUICK-260901-wom]

coverage:
  - id: D1
    description: "trunk.yaml and buf.yaml repaired so both tools can actually run (Go pin raised, buf-lint/golangci-lint gated off, bad_proto fixture excluded)"
    requirement: "QUICK-260901-wom"
    verification:
      - kind: other
        ref: "buf build -o /dev/null && buf breaking --against '.git#branch=main' --against-config buf.yaml"
        status: pass
    human_judgment: false
  - id: D2
    description: "Lint workflow added: trunk-action hold-the-line check + buf breaking, gated on pull_request only"
    requirement: "QUICK-260901-wom"
    verification:
      - kind: other
        ref: "python3 YAML-shape assertion + buf breaking dry run (see plan Task 2 verify)"
        status: pass
    human_judgment: true
    rationale: "The trunk CLI itself is not installed locally (sudo-gated installer, no password in this sandbox); trunk-action's actual behavior on a PR can only be confirmed by the first live CI run."
  - id: D3
    description: "go.yml hardened (read-only token, concurrency group, job timeout, no fetch-depth:0, gotestsum pinned to v1.13.0 with replay form)"
    requirement: "QUICK-260901-wom"
    verification:
      - kind: other
        ref: "python3 YAML assertion (permissions/concurrency/timeout/no fetch-depth/gotestsum@v1.13.0) + local gotestsum replay against real test_results.json"
        status: pass
    human_judgment: false
  - id: D4
    description: "renovate.json consolidated to a single valid config with grouped gomod rules, patch automerge, github-actions automerge, and non-automerging labelled vulnerability alerts; .github/renovate.json deleted"
    requirement: "QUICK-260901-wom"
    verification:
      - kind: other
        ref: "jq schema assertion (see plan Task 3 verify) + test ! -e .github/renovate.json"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-01
status: complete
---

# Quick Task 260901-wom: CI Hardening — Enforce Lint via Trunk Gate Summary

**Added a PR-gated Lint workflow (trunk hold-the-line + buf breaking), repaired the trunk/buf configs that would have made it land red, hardened go.yml's token scope/concurrency/timeout/reporter pin, and consolidated Renovate onto one config that groups Go module bumps instead of shipping them as mutually-conflicting PRs.**

## Performance

- **Duration:** 25 min
- **Tasks:** 3/3 completed
- **Files modified:** 6 (1 created, 4 modified, 1 deleted)

## Accomplishments

- `.github/workflows/lint.yml` now gates every PR to `main` with trunk's hold-the-line check and `buf breaking --against main`
- `.trunk/trunk.yaml` and `buf.yaml` are runnable again: the Go runtime pin matches the go1.25.8 module floor, `buf-lint` is disabled (73 unfixable-without-wire-break findings), `golangci-lint`'s dead v1.55.2 pin is removed, and `buf.yaml` excludes the deliberate `bad_proto` syntax-error fixture
- `go.yml` runs with `contents: read`, a cancel-in-progress concurrency group, a 20-minute job timeout, and a version-pinned test reporter (`gotestsum@v1.13.0`) instead of resolving `@latest` at build time
- Renovate now reads exactly one config (`.github/renovate.json`, which ran `bazel run gazelle` against a `deps.bzl` that no longer exists, is deleted); root `renovate.json` groups all `gomod` updates into one PR, auto-merges patch-level gomod and github-actions bumps, and never auto-merges vulnerability alerts

## Task Commits

1. **Task 1: Repair the trunk and buf configs so they can actually run** - `67e86fa` (fix)
2. **Task 2: Add the Lint workflow — trunk check plus buf breaking** - `20a5961` (feat)
3. **Task 3: Harden go.yml and consolidate the renovate config** - `47b85cd` (chore)

**Plan metadata:** commit made separately by the orchestrator (docs artifacts excluded from this executor's commits per constraints)

## Files Created/Modified

- `.github/workflows/lint.yml` - New PR-only workflow: `trunk` job (checkout + trunk-io/trunk-action@v1) and `buf-breaking` job (checkout, fetch+pin local `main`, buf-setup-action, `buf breaking --against-config buf.yaml`)
- `.trunk/trunk.yaml` - Go runtime raised 1.21.0 → 1.25.8, gofmt raised to match, `buf-lint@1.28.1` moved to `lint.disabled`, `golangci-lint@1.55.2` removed entirely
- `buf.yaml` - Added `build.excludes: [utils/testdata, examples, node_modules]`
- `.github/workflows/go.yml` - Added `permissions: contents: read`, a concurrency group, `timeout-minutes: 20`, dropped `fetch-depth: 0`, pinned the test-report step to `gotestsum@v1.13.0` using the `--raw-command` replay form
- `renovate.json` - Rewritten on `config:recommended` with 3 `packageRules` (gomod group, gomod patch-automerge, github-actions automerge) and a non-automerging labelled `vulnerabilityAlerts` block
- `.github/renovate.json` - Deleted (dead bazel `postUpgradeTasks`, and Renovate errors when multiple config files are present)

## Decisions Made

- **golangci-lint removed, not re-pinned.** The plan's preferred path was to install the trunk CLI (`curl -fsSL https://get.trunk.io | bash -s -- -y`) and let `trunk upgrade` resolve a working pin. The installer downloaded fine but requires `sudo` to write `/usr/local/bin/trunk`, and this sandbox has no usable sudo password (`sudo -n true` fails, and `/usr/local/bin` itself is not user-writable). Per the plan's explicit fallback: `cli.version` stayed at 1.19.0, plugins `ref` stayed at v1.4.2, and `golangci-lint@1.55.2` was removed from `lint.enabled` outright rather than guessing a compatible pin — no v1-line pin can typecheck a go1.25.8 module, and `go build`/`go vet`/`go test` in `go.yml` already cover Go correctness. It can return once someone with a local trunk install re-resolves the pin (likely onto the `golangci-lint2` linter name per the plan's guidance).
- **buf-lint disabled, not deleted.** Kept as a `lint.disabled` entry (with the version pin intact) rather than removed, so it stays visible as a known future item without becoming an active gate.
- **gotestsum replay form validated before committing**, as the plan required. Generated a real `test_results.json` via `go test -v ./utils/... -json`, ran `go run gotest.tools/gotestsum@v1.13.0 --format testdox --raw-command -- cat test_results.json` against it locally, confirmed correct testdox output for all 13 tests and a clean exit code, then applied the same form to `go.yml`.

## Deviations from Plan

None — plan executed exactly as written, including its pre-planned fallback branch for the golangci-lint version strategy (the plan anticipated this exact sudo-blocked scenario and specified the fallback explicitly).

## Issues Encountered

- **trunk CLI could not be installed.** `curl -fsSL https://get.trunk.io | bash -s -- -y` downloaded the launcher but its install step shells out to `sudo` to write `/usr/local/bin/trunk`; this sandbox has no interactive TTY and no passwordless sudo (`sudo: a password is required`, confirmed with `sudo -n true`). Followed the plan's fallback path exactly (see Decisions above). **This means the trunk portion of `lint.yml` is validated only by static YAML-shape checks and the config's own `python3` sanity assertions — the trunk-action's actual behavior on a real PR (whether it resolves its upstream ref cleanly, whether `fetch-depth: 0` turns out to be needed, whether any of the still-enabled linters choke on this repo) is untested locally and will only be known on the first live CI run.** The plan anticipated this and explicitly scoped the honesty constraint around it.
- Everything else (`buf build`, `buf breaking`, YAML/JSON shape assertions, `go build ./...`, and the gotestsum replay form) was verified locally with tools that ARE installed (`buf` 1.61.0, `python3`+PyYAML, `jq`, `go` 1.25.8) and all passed.

## User Setup Required

None - no external service configuration required. (Renovate itself needs no manual setup beyond the config file; it's already installed as a GitHub App per the existing repo setup.)

## Next Phase Readiness

- The Lint workflow will run for the first time on this plan's own PR (stacked on `ci/hardening` → `deps/security-bumps` → PR #513) — that first run is the real validation of the trunk config and should be checked before merge.
- Follow-up opportunities left deliberately out of scope, per the plan's hard limits: re-enabling `buf lint` (needs a breaking-change migration for 73 findings), re-pinning `golangci-lint` (needs someone with local trunk access), fixing the go vet backlog and the flaky `TestProtoconfKVAgentRollout_SubscribeForConfig` test, SHA-pinning actions, and adding a macOS matrix.
- No `.proto` file, no Go source, and no branch was touched, matching the plan's success criteria.

---
*Phase: quick-260901-wom*
*Completed: 2026-09-01*

## Self-Check: PASSED

- FOUND: .github/workflows/lint.yml
- FOUND: .trunk/trunk.yaml (modified as described)
- FOUND: buf.yaml (modified as described)
- FOUND: .github/workflows/go.yml (modified as described)
- FOUND: renovate.json (modified as described)
- MISSING (expected): .github/renovate.json (deleted intentionally)
- FOUND commit 67e86fa: fix(ci): repair trunk and buf configs so they can run
- FOUND commit 20a5961: feat(ci): add Lint workflow — trunk hold-the-line check plus buf breaking
- FOUND commit 47b85cd: chore(ci): harden go.yml and consolidate renovate config
