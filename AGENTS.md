# Grafana Cloud Vending Machine

A portable, public reference implementation of a Grafana Cloud stack vending machine, built on
Crossplane reconciliation and the Grafana Crossplane provider. It carries no source-environment
identity, no credentials and no live requests. Examples are inert by construction.

`enabled/` is the only Argo-watched live-request directory, and it starts empty. `scripts/validate.sh`
asserts the ApplicationSet watches `enabled/*` and nothing else, so adding a second watch path breaks
the gate deliberately.

## The gate

`just check` runs `scripts/validate.sh`, byte for byte the same script the hosted
`Validate public reference` workflow runs. It is the whole local gate and must pass before you commit.

Completion claims carry evidence: the completing SHA and the hosted validation run ID. A green local
run alone is not `Done`.

## The publication constraint

`scripts/public-release-scan.sh` (also `just public-release-scan`, and the first stage of the gate)
scans the working tree and every reachable Git revision for source-environment identifiers, token
prefixes, private endpoints, key material and forbidden filenames. Because it scans history, a banned
literal that reaches a commit is not fixed by a later commit that removes it, and rewriting published
history is not an option here.

The trap that catches agents specifically is absolute local paths: the scan rejects the macOS
home-directory prefix case-sensitively, and tooling instructions, hook tests and pasted command lines
carry them by default. Derive paths from `git rev-parse --show-toplevel` or use relative paths. Never
hard-code one, in this file included.

## Task tracking

Work is tracked with Backlog.md in `backlog/`, committed to git. `backlog doc list --plain` shows the
two operating-model documents: **Agent fan-out protocol (canonical)**, imported verbatim from the
upstream sourcebook and re-imported in the same change whenever upstream moves, and **Wave operating
model**, this project's own lane conventions, recurring defects, exclusive publishing resource and
run-end. Read both before designing a wave.

`backlog/` is committed to git and is inside the publication scan, so tasks, docs and decisions must
never contain real account identifiers or personal data: no email addresses, handles, usernames,
account IDs, stack slugs, device names, addresses or coordinates. Write the shape, not the instance.
Aggregate counts, timings and structural findings are fine. A leak here fails the scan permanently.

Backlog CLI traps beyond the global "drive it through the CLI" rule:

- `--notes` and `--plan` bare **silently replace** the whole section and exit 0, destroying another
  session's writes. Open upstream bug. Use `--append-notes` and `--append-plan`.
- Hand-editing tracker markdown breaks the HTML-comment section markers, and the section is silently
  dropped at exit 0: the data stays in the file, invisible to the CLI, until the next write destroys
  it. There is no repair command; `backlog doctor` only fixes duplicate task IDs. A global `PreToolUse`
  hook denies both this and the bare `--notes`/`--plan` forms.
- `backlog/config.yml` is the deliberate exception and is edited by hand, because list-valued keys
  cannot be set through `backlog config set`.
- Finalize in one call so an interrupted run cannot leave finished work looking unfinished:
  `backlog task edit GCV-0007 --check-ac 1 --check-ac 2 -s Done`.
- Two agents must never edit the same task. The v1.50.x fix covers the edit funnel but not reorder,
  draft saves, the TUI path, `doc update` or decision updates.
- Statuses are `To Do`, `In Progress`, `Parked`, `Done`. `Parked` means attempted, blocked, and left
  with a concrete resume boundary. It is not `To Do`, and flattening it loses the most valuable thing
  a long run produces.

## Deeper references

- `archive/README.md` - read before citing pre-Backlog history; the closed GitHub issues live there
  and are indexed by the **Closed GitHub issues (pre-Backlog history index)** Backlog doc.

<!-- BACKLOG.MD GUIDELINES START -->
<!-- backlog.md-instructions-version: 1.50.1 -->
<CRITICAL_INSTRUCTION>

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**For every user request in this project, run `backlog instructions overview` before answering or taking action.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:
- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- BACKLOG.MD GUIDELINES END -->

## Releases

Release automation runs on pushes to `main` through release-please. Use Conventional Commits so
release notes and version bumps describe compatibility changes accurately:

- `feat:` produces a minor release.
- `fix:` and `perf:` produce patch releases.
- Add `!` after the type or scope, or add a `BREAKING CHANGE:` footer, for a breaking release. The
  initial manifest version is `0.1.0`, and pre-1.0 breaking bumps are configured to advance to
  `1.0.0`.
- `docs`, `refactor`, `test`, `build`, `ci`, and `chore` are categorized according to the release
  configuration, with non-user-facing categories hidden where configured.

Release token minting uses a short-lived repository-scoped broker token. If minting fails, check
the broker/OpenBao infrastructure rather than treating the failure as a source-code validation
result or adding a long-lived credential.
