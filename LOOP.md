# LOOP.md

This file holds loop-specific facts for this repository, so loop goals cite it instead of restating
them. Read it at loop preparation.

## Gates and commands

- `just check` runs `scripts/validate.sh`, byte-for-byte the same script the hosted `Validate public
  reference` CI workflow runs. It is the whole local gate and must pass before committing (AGENTS.md
  "The gate"; justfile: `check`).
- `just public-release-scan` (also the gate's first stage) scans the working tree and every reachable
  git revision for source-environment identifiers, token prefixes, private endpoints, key material
  and forbidden filenames. A banned literal that ever reached a commit is a permanent failure because
  published history is not rewritten (AGENTS.md "The publication constraint").
- `just setup` verifies the local toolchain (`rg`, `ruby`, `kubectl`, `go`) and that
  `KUBEBUILDER_ASSETS` points at a matching pinned `kube-apiserver`/`etcd` (justfile: `setup`).
- `just envtest` downloads and checksum-verifies the pinned envtest API-server assets into a cache
  resolved to sit outside the repository tree — otherwise both the gate's YAML walk and the
  publication scan see the vendored binaries. Never transcribe the published checksum by hand
  (justfile: `envtest`).
- `just envtest-process-check` / `just envtest-process-reap [max_age_seconds]` report or reap leaked
  repository-pinned envtest processes and temp directories over an age ceiling (justfile).
- `just fmt` / `fmt-check` — gofmt plus `just --fmt`; `just lint` — `go vet` plus a `go.mod`/`go.sum`
  tidiness check; `just test [filter]` — race-enabled Go tests with coverage (justfile).
- `just image [tag]` builds the composition-function container locally for the host platform; never
  pushes (justfile: `image`).
- Completion claims carry evidence: the completing SHA and the hosted validation run ID. A green
  local run alone is not "Done" (AGENTS.md "The gate").

## Release rules

- Release automation runs release-please on pushes to `main`, driven by Conventional Commits
  (`feat:` minor, `fix:`/`perf:` patch, a `!` or `BREAKING CHANGE:` footer breaking; pre-1.0 breaking
  bumps advance straight to `1.0.0`) (AGENTS.md "Releases").
- **Per-loop cap: at most one release-please pull request is merged, and only as the loop's last
  publishing act when that loop's goal explicitly authorises it.** The default across loops is to
  leave the pull request untouched for the owner and report its number, head SHA and check state as
  found at close (`goal-2026-09-24-loop18.md`: "squash-merge exactly one release-please pull request
  under section 7.5"; `goal-2026-09-24-loop19.md` §0 item 2: "leave the release-please pull request
  for the owner. The loop makes no pull-request write of any kind"; `goal-2026-09-24-loop20.md` and
  `goal-2026-09-24-loop21.md`: "Do not touch #<PR> or any successor release-please PR").
- No comment, label, review, rebase request or Renovate-PR merge is authorised outside that single
  release merge (`goal-2026-09-24-loop18.md` §"External-write authority").
- Release token minting uses a short-lived, repository-scoped broker token. If minting fails, treat
  it as a broker/OpenBao infrastructure problem, not a source-code validation result, and never add a
  long-lived credential (AGENTS.md "Releases").

## Environments and credential conventions

- The repository carries no source-environment identity, no credentials and no live requests by
  design; examples are inert. `enabled/` is the only Argo-watched live-request directory and starts
  empty (AGENTS.md).
- Loop lanes are correspondingly network-read-only at most: a proof/review lane may read pinned
  upstream provider/module source at an exact revision, but "no live Grafana Cloud contact, no live
  cluster contact, no estate action" (`goal-2026-09-24-loop18.md` §"External-write authority"; the
  same scoping recurs for the review lanes in loops 19-21).
- Cluster tooling is always scoped: every invocation uses `KUBECONFIG=/dev/null` or the envtest
  kubeconfig, with `KUBEBUILDER_ASSETS` set from `just envtest`'s own printed export; an unset
  `KUBEBUILDER_ASSETS` is a known environment gap, not a silent default (justfile: `setup`;
  `goal-2026-09-24-loop18.md` §6.7).
- `XDG_CACHE_HOME` (or its default) must resolve outside the repository tree before envtest assets
  are written there (justfile: `envtest`).

## Standing route exceptions

- **Task GCV-0075 (provider-boundary / same-token-handoff proof, and its independent review).**
  - Lane kind: a single read-only SECURITY/design-judgement lane — no implementation, no live or
    estate action, network limited to reads of pinned public upstream source.
  - Scope: route to `gpt-6-astra` at `high`, one-shot per loop (no correction round, no stacking of
    attempts). Re-granted by the owner at preparation across four consecutive units — wave 17 lane F,
    loop 19 lane P, loop 20 lane H, loop 21 lane R2a — each time despite that loop's own goal calling
    it "not a precedent" and despite the fan-out protocol's default "never automatically launch
    Astra/high ... only a new explicit operator instruction" (`goal-2026-09-24-loop21.md` line 195:
    "GCV-0075, Astra/high one-shots | wave 17 F, loop 19 P, loop 20 H | loop 21 R2a, one-shot review,
    no correction round | Each needs a new explicit owner instruction"; corroborated in
    `goal-2026-09-24-loop19.md` §0 item 1 and `goal-2026-09-24-loop20.md` §0 item 1;
    `work.md` D9, lines 316-326).
  - Expiry: none given by any source. Treat as standing until revoked by Rob.
  - Fallback on refusal: see Known traps below.

## Known traps

- **Astra refuses a SECURITY-framed review under provider `cyber_policy`, not under an effort or
  route limit.** An Astra turn framed around constructing or describing attacks is flagged
  (`codex_error_info: cyber_policy`); the same design/handoff proof on Astra/high framed as a neutral
  design question is not. Frame a SECURITY or REVIEW brief as a correctness, concurrency and
  ownership review, with no "attack" or "exploit" wording. On a `cyber_policy` refusal, do not retry
  the same thread; fall back once, one-shot, to `gpt-6-sol` at `high` with the identical brief; if
  that also fails, park the review for the owner — there is no third route
  (`goal-2026-09-24-loop21.md` §5.3, lines 170-174; `work.md` D8, lines 300-313).
- **Absolute macOS home-directory paths fail the publication scan case-sensitively**, and tooling
  instructions, hook tests and pasted command lines carry them by default; derive paths from
  `git rev-parse --show-toplevel` or use relative paths, everywhere including this file (AGENTS.md
  "The publication constraint").
- **A worktree isolates files only**, never the Go build/module caches, the envtest assets under
  `$HOME`, the host process table or Git refs; see Resource mutexes (`goal-2026-09-24-loop18.md`
  §3.2).
- **`backlog task edit --notes`/`--plan` bare silently replaces the whole section and exits 0**,
  destroying another session's writes; use `--append-notes`/`--append-plan` (AGENTS.md "Backlog CLI
  traps").

## Cross-harness eligibility

None approved; ask at preparation.

## Resource mutexes

- **The host's envtest process table and its pinned assets path** are shared across lanes and the
  root's own gate runs; a lane needing isolation works from a lane-private copy instead of the shared
  path (`goal-2026-09-24-loop18.md` §"Shared resource: the host's envtest process table").
- **Publication** (commit/push to `main`, the single release-please merge, tag and release creation)
  is root-only; a worktree grants a lane no commit-to-`main` or publication authority
  (`goal-2026-09-24-loop18.md` §3.2).

## Grafana stacks

The repository is inert by design: no live Grafana Cloud contact, and no stack, real
or example, is ever named or reached by a loop (AGENTS.md; `goal-2026-09-24-loop18.md`
§"External-write authority": "No live Grafana Cloud contact").
