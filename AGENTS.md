# Grafana Cloud Vending Machine — contributor and agent instructions

This is the canonical instruction file. `CLAUDE.md` imports it, so Claude Code and Codex read the
same thing and the two cannot drift apart.

## What this repository is

A portable, public reference implementation of a Grafana Cloud stack vending machine, built on
Crossplane reconciliation and the Grafana Crossplane provider. It carries **no source-environment
identity, no credentials, and no live requests.** Examples are inert by construction.

- `platform/` — the product: namespaced APIs, pipeline Compositions, the repository-owned Go
  composition function, provider wiring, composition RBAC.
- `examples/catalog/*` — inert consumer examples, each a renderable Kustomize base with a README.
- `enabled/` — the only Argo-watched live-request directory. Starts empty and inert.
- `deploy/` — optional installation and GitOps integration.
- `docs/` + `README.md` — the published documentation site and the single comprehensive guide.
- `archive/` — the pre-Backlog GitHub Issues archive. See `archive/README.md`.

## The gate

```bash
./scripts/validate.sh
```

One command, and it is the whole local gate: the public-release scan, `gofmt`, `go mod tidy` with a
`git diff --exit-code` on `go.mod`/`go.sum`, race-enabled tests with coverage, `go vet`, a YAML parse
of every tracked YAML document, four Kustomize renders, example-README coverage, and the
ApplicationSet watch-path assertion. The hosted `Validate public reference` workflow runs the same
script.

**Completion claims carry evidence: the completing SHA and the hosted validation run ID.** A green
local run alone is not `Done`.

## The publication constraint

`scripts/public-release-scan.sh` scans the working tree **and every reachable Git revision** for
source-environment identifiers, token prefixes, private endpoints, key material and forbidden
filenames. Because it scans history, a banned literal that reaches a commit is **not** fixed by a
later commit that removes it, and rewriting published history is not an option here.

The trap that catches agents specifically is **absolute local paths** — the scan rejects the macOS
home-directory prefix case-sensitively, and tooling instructions, hook tests and pasted command
lines carry them by default. Anything committed here derives its paths from `git rev-parse
--show-toplevel` or `CLAUDE_PROJECT_DIR`, or uses relative paths. Never hard-code one.

Before committing, run the scan. It is the first stage of the gate, so `./scripts/validate.sh`
covers it.

## Task tracking

Work is tracked with Backlog.md in `backlog/`, committed to git. Two documents carry the operating
model — `backlog doc list --plain` shows them:

- **Agent fan-out protocol (canonical)** — the campaign model, imported verbatim. Read it before
  designing a wave. When the upstream sourcebook changes, re-import this copy in the same change.
- **Wave operating model** — this project's own rules, recurring defects, lane conventions, the
  exclusive publishing resource, and run-end. Read it before starting work.

A third, **Closed GitHub issues — pre-Backlog history index**, indexes the work that predates the
tracker.

### Rules that are not negotiable

**`backlog/` is committed to git, so tasks, docs and decisions must never contain real account
identifiers or personal data** — no email addresses, handles, usernames, account IDs, stack slugs,
device names, addresses or coordinates. Write the shape, not the instance. Aggregate counts, timings
and structural findings are fine. This is easy to break by accident precisely because a tracker
feels private, and here it also fails the publication scan permanently. Sweep before committing:

```bash
./scripts/public-release-scan.sh
```

**Never use `--notes` or `--plan` bare.** They *silently replace* the whole section, destroying
another session's writes with no warning and exit 0. This is an open upstream bug, not a
misunderstanding. Use `--append-notes` and `--append-plan`. A `PreToolUse` hook in
`.claude/hooks/backlog-guard.py` denies the unsafe forms rather than trusting anyone to remember.

**Never hand-edit task, draft, doc, decision or milestone markdown.** Section boundaries are
HTML-comment markers; break one and the section is *silently dropped* at exit 0 — the data stays in
the file but is invisible to the CLI until the next write destroys it for real. There is no repair
command; `backlog doctor` only fixes duplicate task IDs. The same hook denies these edits.
`backlog/config.yml` is the deliberate exception and may be edited by hand, because list-valued keys
cannot be set through `backlog config set`.

**Finalize in one call**, so an interrupted agent cannot leave finished work looking unfinished:

```bash
backlog task edit GCV-0007 --check-ac 1 --check-ac 2 -s Done
```

**Never let two agents edit the same task.** The v1.50.x fix covers the edit funnel but not reorder,
draft saves, the TUI path, `doc update` or decision updates.

**Statuses are `To Do`, `In Progress`, `Parked`, `Done`.** `Parked` means attempted, blocked, and
left with a concrete resume boundary — it is not `To Do`, and flattening it loses the most valuable
thing a long run produces.

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
