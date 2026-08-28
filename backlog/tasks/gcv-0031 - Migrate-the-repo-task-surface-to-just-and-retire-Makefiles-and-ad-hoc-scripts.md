---
id: GCV-0031
title: Migrate the repo task surface to just and retire Makefiles and ad-hoc scripts
status: To Do
assignee: []
created_date: '2026-08-28 19:20'
labels: []
dependencies: []
priority: medium
type: chore
ordinal: 31000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
# Migrate the repo task surface to just; no Makefile exists, no script is absorbed

## 1. Outcome

A top-level `justfile` exists at the repo root exposing the fleet-mandatory recipe vocabulary
(`default`, `setup`, `fmt`, `fmt-check`, `lint`, `test`, `check`) plus repo-specific convenience
recipes (`public-release-scan`, `build`, `image`). `just check` is byte-for-byte what
`.github/workflows/validate.yml` runs (`./scripts/validate.sh`) — there is zero drift between the
local gate and CI because `check` delegates to that script rather than reimplementing its logic.
Both existing shell scripts (`scripts/validate.sh`, `scripts/public-release-scan.sh`) are **kept as
files, unmodified**, and reachable only through `just` recipes — nobody types `./scripts/foo.sh`
again. `AGENTS.md`, `README.md`, `docs/troubleshooting.md`, `docs/security.md`, and
`backlog/config.yml`'s `definition_of_done` point at `just check` / `just public-release-scan`
instead of the bare script paths. No Makefile exists in this repo, so there is no Makefile deletion
step. CI's `validate.yml` gains a `setup-just` step and its one `run:` line becomes `run: just
check`; nothing else in any workflow changes.

**Read this whole document before touching the repo. It is self-contained; make no design
decisions of your own — every choice below (including the KEEP calls) is already made and
justified.**

## 2. The complete justfile

Write this to `justfile` at the repo root. Adjust only where a `require()` check surfaces a real
local gap (e.g. an actually-different `rg`/`kubectl`/`ruby` binary name), and where the
`extractions/setup-just` SHA/`just-version` need reconciling with whatever the fleet has pinned
elsewhere (see §5 and §9 — do not invent a SHA).

```just
set shell := ["bash", "-euo", "pipefail", "-c"]

# show the task surface
default:
    @just --list

# verify the local toolchain the repo's recipes assume is present
[group('check')]
setup:
    @require rg
    @require ruby
    @require kubectl
    @require go
    cd platform/function && go mod download

# format Go source and the justfile in place
[group('check')]
fmt:
    cd platform/function && gofmt -l -s -w .
    just --fmt

# verify formatting without mutating; never modifies files
[group('check')]
[no-exit-message]
fmt-check:
    cd platform/function && test -z "$(gofmt -l .)" || (cd platform/function && gofmt -l . && exit 1)
    just --fmt --check

# go vet plus the go.mod/go.sum tidiness check CI enforces
[group('check')]
[no-exit-message]
lint:
    cd platform/function && go vet ./...
    cd platform/function && go mod tidy
    cd platform/function && git diff --exit-code -- go.mod go.sum

# race-enabled Go test suite with coverage; optional filter=<pattern> narrows by -run
[group('check')]
[no-exit-message]
test filter="":
    cd platform/function && go test -race -cover {{ if filter != "" { "-run " + filter } else { "" } }} ./...

# scan the working tree and reachable Git history for source-environment identifiers
[group('check')]
public-release-scan:
    ./scripts/public-release-scan.sh

# THE GATE — exactly what CI runs; byte-for-byte the same script, zero drift
[group('check')]
[no-exit-message]
check:
    ./scripts/validate.sh

# build the composition function binary for the host platform
[group('build')]
build:
    cd platform/function && go build -trimpath -o bin/function-server .

# build the function's runtime container image locally (no push)
[group('build')]
image tag="function-grafana-vending:dev":
    docker buildx build --platform linux/amd64 --target image -t {{ tag }} platform/function
```

**Why `check` is a single delegating line and not `check: fmt-check lint test public-release-scan`:**
`scripts/validate.sh` is KEPT (see §4) and does strictly more than the sum of the other recipes — it
also runs a YAML syntax parse of every tracked YAML document, four Kustomize renders (`platform`,
`deploy/aws`, `examples/catalog/comprehensive`, `examples/catalog/minimal`), the example-README
coverage check, the package-digest cross-check between each verification Job and its Function/Provider
resource, and the ArgoCD ApplicationSet watch-path assertion. None of that logic is being ported into
recipe bodies (§4 explains why). If `check` only depended on the decomposed recipes it would be a
**subset** of what CI enforces — a violation of the fleet standard's "check must be complete" rule.
Delegating the whole gate to the exact script CI already runs is the only way to guarantee `just
check` and the `Validate reference` CI job never drift apart. `fmt-check`/`lint`/`test`/
`public-release-scan` still exist as fast, separately-runnable recipes for iteration — they are not
redundant, they are conveniences that happen to share command text with what `validate.sh` also runs.

**`fmt-check`'s odd body**: `gofmt -l .` prints nothing (exit varies) on clean input; the
`test -z "$(...)" || (...)` idiom is needed because plain `gofmt -l` never itself returns non-zero on
unformatted files — the check has to be on the emptiness of its output, matching what
`scripts/validate.sh:9-13` already does. Keep this exact idiom; do not "simplify" it to bare
`gofmt -l .` or the recipe stops catching unformatted files.

## 3. Makefile disposition

**No Makefile or GNUmakefile exists anywhere in this repository** (verified: `find . -iname
"Makefile" -o -iname "GNUmakefile"` returns nothing, excluding `vendor/`/`node_modules/` which also
do not exist here). There is nothing to delete and no `git rm` step for this section.

## 4. Script disposition

| Script | Verdict | Recipe | Reasoning |
|---|---|---|---|
| `scripts/validate.sh` | **KEEP** | `just check` (§2, delegates verbatim) | Contains non-trivial control flow (a bash `for` loop over `examples/catalog/*`, two inline `ruby -e` programs with real function definitions doing cross-file consistency checks — see below) and is not a thin sequencer per §6's MIGRATE test. More importantly: it is a **stable, extensively load-bearing identifier** referenced by exact path across 30+ committed historical `backlog/tasks/*.md` records (their `- [ ]`/`- [x]` acceptance criteria literally read `./scripts/validate.sh passes locally`, and several completed tasks' implementation-evidence prose quotes exact script behaviour), `backlog/docs/doc-0002` ("Wave operating model" — states outright *"it is what every other lane is being judged by"* and *"Adding a catalog directory means adding its render to `scripts/validate.sh` too"*), `README.md:749`, `docs/troubleshooting.md:127`, `docs/security.md:93`, `AGENTS.md`, and `backlog/config.yml`'s `definition_of_done`. `AGENTS.md` explicitly forbids hand-editing backlog task/doc markdown, so those historical references cannot and must not be rewritten — absorbing the script would strand dozens of historical acceptance-criteria strings pointing at a file that no longer exists. Wrapping it in one recipe carries zero risk and zero drift; absorbing it would require re-deriving its exact behaviour as `just` recipe bodies (fighting the multi-line-shell-per-line limitation, per §10 of the standard) for no benefit, since nothing downstream needs it gone. |
| `scripts/public-release-scan.sh` | **KEEP** | `just public-release-scan` | Real program: ~165 lines, four bash function definitions (`scan_fixed`, `scan_regex`, `scan_fixed_allowing`, `sift_hits`, `run_search`), an `allowed`-substring-stripping sieve implemented in Perl, careful exit-code discipline (a failed search must exit >1, distinct from a clean "no match" exit 1), and it scans **all of reachable Git history** (`git rev-list --all` then `git grep` per revision), not just the working tree. This is squarely the §6 "real program" / "non-trivial control flow" KEEP case, and it is independently documented as a standalone tool developers are told to run before making a fork public (`docs/security.md:91-98`, `AGENTS.md`'s "Before committing, run the scan" line). It is called both directly (`just public-release-scan`) and internally by `scripts/validate.sh` (unchanged, still calls it by relative path) — do not change that internal call. |

No other tracked scripts exist (`git ls-files | grep -E '\.(sh|bash|zsh|ps1)$'` returns exactly
these two; no `scripts/*.py` or other-language task scripts exist).

## 5. CI changes

### `.github/workflows/validate.yml`

Current (full file, 38 lines) runs one step: `run: ./scripts/validate.sh`, after installing
ripgrep via `apt-get` and Go via `actions/setup-go`.

Exact edit — insert a `setup-just` step after the Go setup step, and change the final step's `run:`
line only:

```yaml
      - name: Set up Go
        uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7
        with:
          go-version: 1.27.0
          cache-dependency-path: platform/function/go.sum

      - name: Set up just
        uses: extractions/setup-just@<pin-exact-sha> # v3
        with:
          just-version: '1.58.0'

      - name: Validate
        run: just check
```

**Do not fabricate the SHA.** `<pin-exact-sha>` must be resolved to the real current commit SHA for
the `extractions/setup-just` tag in use fleet-wide (other rknightion/m7kni repos already mid-migration
pin it — check a sibling repo's just-migrated `justfile`/workflow, or resolve it directly from GitHub
at implementation time) before this lands. A wrong or placeholder SHA breaks CI outright on the next
push.

**Everything else in this file is untouched**: `permissions: contents: read`, the `concurrency`
block, `actions/checkout` with `fetch-depth: 0` and `persist-credentials: false`, the ripgrep
`apt-get install` step (still needed — `scripts/public-release-scan.sh` still `require`s `rg` on
PATH and runs inside the same job), the job name `validate`, and the workflow name `Validate public
reference`. There is no `ci-success` aggregator in this repo (single-job workflow) — do not invent
one.

### `.github/workflows/publish-function.yml`

**Do not touch.** This is a matrix-build, sign-and-publish pipeline (Go test/vet in a `test` job,
QEMU + Buildx multi-arch image build, `crossplane xpkg build`/`push`, Cosign keyless signing via
OIDC) that is explicitly out of scope per §8 of the standard ("do not fold release-please, CodeQL,
zizmor, actionlint, scorecard, dependency-review or container-publish workflows into `just`" — this
is the container-publish case). Its `test`/`Vet` steps duplicate commands that also exist as
`just lint`/`just test` bodies; that overlap is intentional and harmless — leave both as-is. Do not
convert its `uses:` steps (`docker/build-push-action`, `docker/login-action`,
`sigstore/cosign-installer`, `docker/setup-buildx-action`, `docker/setup-qemu-action`) into `run:
just` calls.

### `.github/workflows/trigger-docs-sync.yml`

**Do not touch.** GitHub-native repository-dispatch trigger with an OpenBao-broker-minted token; no
build/test/lint/generate/validate logic to migrate. Per `[[release-please-pat.md]]`/
`[[docs-sync-pat.md]]` conventions already governing this repo (see the `broker-token` step and its
comments) — this is infrastructure plumbing, not task surface.

## 6. Docs and agent-contract changes

Every exact hit for `./scripts/validate.sh` / `scripts/validate.sh` / `scripts/public-release-scan.sh`
as a **command a human or agent is told to type**, and its replacement. Prose that *describes* what
the script does (its internal behaviour) is factually unchanged by this migration and does not need
editing — only the invocation changes.

| File | Line(s) | Current | Change to |
|---|---|---|---|
| `AGENTS.md` | `## The gate` code block (~line 23) | ` ```bash\n./scripts/validate.sh\n``` ` | ` ```bash\njust check\n``` ` — keep the prose paragraph below it describing what the gate does verbatim (still accurate; `just check` runs the identical script). |
| `AGENTS.md` | ~line 47 "Before committing, run the scan." code fence | (verify exact fence content around this line; it currently instructs running the scan) | change the invocation to `just public-release-scan` |
| `AGENTS.md` | Add a new `## Task interface` section | — | Insert the standard §9 block verbatim (below), placed near `## The gate` since both describe how to run checks. |
| `README.md` | line 749 code fence | ` ```bash\n./scripts/validate.sh\n``` ` | ` ```bash\njust check\n``` ` |
| `docs/troubleshooting.md` | line 127 code fence, under "## Running the validation gate" | ` ```bash\n./scripts/validate.sh\n``` ` | ` ```bash\njust check\n``` ` |
| `docs/security.md` | line 93, "`scripts/validate.sh` runs `scripts/public-release-scan.sh`, which scans…" | prose reference | Leave the prose describing behaviour; it is accurate regardless of entry point. Only add, if a command example exists nearby, `just public-release-scan` as the way to run it standalone — check the surrounding paragraph (lines ~91-99) for any bare invocation to convert. |
| `backlog/config.yml` | `definition_of_done` | `["./scripts/validate.sh passes locally", "hosted Validate workflow passes on the completing commit"]` | `["just check passes locally", "hosted Validate workflow passes on the completing commit"]` — this file is the one deliberate hand-edit exception per `AGENTS.md` ("`backlog/config.yml` is the deliberate exception and may be edited by hand"). Do not touch it through the `backlog` CLI. |
| `backlog/docs/doc-0002` ("Wave operating model") | multiple (`## The gate` section, "Adding a catalog directory…", "Lane conventions" bullet on `scripts/validate.sh`) | references the script by path as the gate and as an integration file with a single owner | **Do not hand-edit this file** (durable doc, CLI-managed, HTML-comment section markers). If it needs a `just check`-pointing update, do it through `backlog doc edit doc-0002 --append-notes "..."` or the equivalent doc-update CLI verb — check `backlog doc --help` for the exact update flags before touching it. Prefer leaving it referencing `scripts/validate.sh` by name (still true — the file still exists and does the same thing) and treat "the gate" and "`just check`" as synonyms; only touch this file if a later reviewer specifically asks for it, since it is explicitly called out as single-owner during a wiring pass, not a MIGRATE-lane edit. |
| `backlog/tasks/*.md` (30+ historical files) | acceptance criteria and implementation-evidence prose quoting `./scripts/validate.sh` | historical record | **Do not touch.** These are closed or open historical task records; `AGENTS.md` forbids hand-editing task markdown, and rewriting history here is explicitly the reason `scripts/validate.sh` is KEPT rather than absorbed (§4). Future *new* tasks may phrase their acceptance criteria as `just check passes locally` going forward — that is a matter for whoever authors the next task, not this migration. |

### `AGENTS.md` — Task interface section to add

Insert this block (standard §9, verbatim) into `AGENTS.md`, near the existing `## The gate` section:

```markdown
## Task interface

This repo's task surface is a `justfile`. Discover it, don't guess it:

    just --list                        # human-readable
    just --dump --dump-format json     # machine-readable
    just --show <recipe>               # what a recipe actually runs

- `just check` is the full gate and is exactly what CI enforces. It must pass before you commit.
- Prefer `just <recipe>` over the underlying tool. If you are typing `go test`, you want `just test`.
- Run `just` with stdin from /dev/null. This repo defines no `[confirm]` recipes today, but if one is
  added later, stop and ask before running it; never pass `--yes` or `JUST_YES=1`.
- If a task you need does not exist, add a recipe with a `#` doc comment and a `[group(...)]`
  rather than running a bare command.
```

## 7. `backlog/config.yml`

Exact new `definition_of_done` line (hand-edit this file only — it is the documented exception):

```yaml
definition_of_done: ["just check passes locally", "hosted Validate workflow passes on the completing commit"]
```

Nothing else in `backlog/config.yml` (statuses, `task_prefix: "gcv"`, `zero_padded_ids: 4`, etc.)
changes.

## 8. Order of work

1. Write `justfile` at the repo root exactly as in §2 (adjust only the `setup-just` SHA per §5's
   warning, and re-verify `require` targets resolve on the machine actually running `just setup`).
2. Run `just --fmt --check` — fix formatting if it fails (run `just --fmt` once, re-check).
3. Run `just --list` and confirm all seven mandatory recipes plus `public-release-scan`, `build`,
   `image` show with correct groups and doc comments.
4. Run `just check` locally and confirm it is identical in behaviour to running
   `./scripts/validate.sh` directly (same output, same exit code) — it should be, since it is the
   same script.
5. Run `just fmt-check`, `just lint`, `just test`, `just public-release-scan` individually and
   confirm each passes standalone (this proves the convenience recipes are not silently broken
   even though `check` does not depend on them).
6. Update `.github/workflows/validate.yml` per §5, with a real resolved SHA for
   `extractions/setup-just`. Push and confirm the hosted `Validate public reference` run is green.
7. Update `AGENTS.md`, `README.md`, `docs/troubleshooting.md`, `docs/security.md` per §6.
8. Hand-edit `backlog/config.yml`'s `definition_of_done` per §7.
9. Do not touch `backlog/docs/doc-0002` or any `backlog/tasks/*.md` file (§6 — explicitly out of
   scope; historical/single-owner).
10. There is no deletion step — no Makefile exists, and both scripts are KEPT. The migration is
    entirely additive (`justfile`) plus small pointer edits (docs, CI, `definition_of_done`).

## 9. Traps specific to this repo

- **`scripts/validate.sh` does its own `cd` internally per section** — it starts with
  `cd "$(git rev-parse --show-toplevel)"`, then uses a subshell `( cd platform/function; ... )` for
  the Go steps. This means `just check`'s single-line delegation (`./scripts/validate.sh`) is safe to
  run from any working directory `just` itself starts recipes from (`just` always runs recipe bodies
  from the directory containing the `justfile`, i.e. repo root) — do not add a redundant `cd` before
  it.
- **`fmt-check`'s `gofmt -l` idiom** (see §2 note) — do not simplify it to a bare `gofmt -l .`
  exit-code check; `gofmt -l` alone does not set a non-zero exit status for unformatted files, only
  for parse errors. The `test -z "$(...)" || (...)` shape is required and is what
  `scripts/validate.sh:9-13` already relies on.
- **`go mod tidy` inside `lint` mutates `go.mod`/`go.sum` on disk** before the `git diff --exit-code`
  check runs — this matches existing behaviour in `scripts/validate.sh:17-18` (same pattern: tidy,
  then diff-check) and is intentional, not a bug to "fix" by reordering. A dirty tree after `just
  lint` locally is expected and correct signal, not a regression.
- **The `filter` param on `test`**: `go test ./...`'s `-run` flag needs a value; the just
  conditional-expression form (`{{ if filter != "" { "-run " + filter } else { "" } }}`) is
  standard, stable `just` syntax (not `--unstable`) — verify with `just --fmt --check` after writing
  it, since a syntax slip here is the most likely place to break `--dump`/`--list` for the whole
  file (§5.8 of the standard: one bad recipe can blind the entire file).
- **No `golangci-lint` config exists** (`find . -iname ".golangci*"` returns nothing) — `lint` is
  intentionally just `go vet` + the mod-tidy diff check, matching what `scripts/validate.sh` and
  `publish-function.yml`'s `test` job both already run. Do not invent a golangci-lint step that
  doesn't exist in this repo's actual CI.
- **`ruby` is used via system Ruby** (`/usr/bin/ruby` on the verifying machine, no Gemfile/bundler)
  inside `scripts/validate.sh` for the YAML-parse, digest cross-check, and ArgoCD watch-path
  assertion — `just setup`'s `require ruby` check is there so an agent gets a clear failure message
  instead of a cryptic script error if Ruby is ever missing from a runner image.
- **`kubectl kustomize` is required by `scripts/validate.sh`**, not `kustomize` standalone — the
  `setup` recipe's `require kubectl` check is correct; do not swap in a `require kustomize` check,
  the script never calls the standalone binary.
- **The Docker `image` recipe target name is literally `image`** (`Dockerfile`'s final stage is
  `FROM ... AS image`) — `--target image` in the recipe is not a placeholder, it is the actual stage
  name; do not rename it.
- **This repo has no `.env`/dotenv usage** (`.gitignore` excludes `.env*` but nothing in the repo
  reads one) — the justfile header correctly omits `set dotenv-load`; do not add it speculatively.
- **`enabled/` contains only `.gitkeep`** and `codex/` is gitignored (`/codex/` line in
  `.gitignore`) — neither needs a recipe or a mention; they are not part of the task surface.

## 10. Out of scope

- `platform/function/*.go` application code — no logic changes, this is a task-surface migration
  only.
- `.github/workflows/publish-function.yml` — container-publish pipeline, explicitly excluded per §8
  of the fleet standard (§5 above explains why in full).
- `.github/workflows/trigger-docs-sync.yml` — GitHub-native dispatch trigger, no build/test/lint
  logic (§5 above).
- `scripts/validate.sh` — KEPT, unmodified, wrapped only (§4).
- `scripts/public-release-scan.sh` — KEPT, unmodified, wrapped only (§4).
- `backlog/docs/doc-0002` ("Wave operating model") and all `backlog/tasks/*.md` — historical/
  single-owner records; not touched by this migration (§6).
- `deploy/`, `platform/apis/`, `platform/rbac/`, `platform/provider/`, `examples/catalog/*` — no
  task-surface changes; these are the product's declarative surface, unrelated to `just`.
- `docs.toml`, the `docs/` content pages other than the two invocation edits in §6 — the
  documentation site's navigation, tags, and page content are unrelated to this migration.
- Any release-please, CodeQL, zizmor, actionlint, scorecard, or dependency-review workflow — none
  currently exist in this repo's `.github/workflows/` (only the three files listed above exist), so
  there is nothing to preserve here beyond noting none were found to accidentally touch.
- No Makefile exists — there is no Makefile-deletion step (§3).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Top-level justfile exists defining default, setup, fmt, fmt-check, lint, test and check with the frozen header (set shell bash -euo pipefail) and no unstable just features
- [ ] #2 just check passes locally and its body is exactly ./scripts/validate.sh — byte-identical to what .github/workflows/validate.yml runs, so there is zero drift between the local gate and CI
- [ ] #3 just --fmt --check passes on the justfile
- [ ] #4 just --list shows a # doc comment and a [group(...)] for every public recipe, including public-release-scan, build and image
- [ ] #5 No Makefile or GNUmakefile is added or needed to be deleted — none existed before this task and confirmedly none exist after
- [ ] #6 scripts/validate.sh and scripts/public-release-scan.sh remain on disk unmodified and are reachable only via just check and just public-release-scan respectively
- [ ] #7 .github/workflows/validate.yml calls just check via a setup-just step pinned to just-version 1.58.0 with a real resolved SHA, with permissions, concurrency, checkout persist-credentials:false and the ripgrep apt-get install step unchanged
- [ ] #8 AGENTS.md, README.md, docs/troubleshooting.md and docs/security.md invoke just check / just public-release-scan instead of the bare script paths, and AGENTS.md gains the Task interface section
- [ ] #9 backlog/config.yml's definition_of_done names just check passes locally instead of ./scripts/validate.sh passes locally, hand-edited per the repo's documented exception
- [ ] #10 No file under backlog/docs or backlog/tasks is hand-edited, and .github/workflows/publish-function.yml and trigger-docs-sync.yml are untouched
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
