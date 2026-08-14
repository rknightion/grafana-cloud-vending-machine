---
id: doc-0002
title: Wave operating model
type: guide
created_date: '2026-08-14 16:36'
updated_date: '2026-08-14 16:36'
---
This document carries **only** what is specific to this repository. The campaign model itself —
run contract and run modes, the routing contract, authority and the thread pool, child lane briefs,
external-contract freezing, the unattended blocker contract, the goal-file template and the
pre-flight checklist — lives in the **Agent fan-out protocol (canonical)** document. Read that
first; this one is the delta. Nothing here restates it, and nothing here that could be pasted into
another project unchanged belongs here.

`backlog doc list --plain` shows both.

## The one constraint that outranks everything

**This repository is a portable public reference. It never touches a live Grafana Cloud stack, a
live cluster, or any source environment.** No lane may create, adopt, import, mutate or delete a
real resource, and no lane may introduce an identifier belonging to one. Examples are inert by
construction and stay that way.

The enforcement is `scripts/public-release-scan.sh`, and its blast radius is what makes this a
standing constraint rather than a lint rule: **it scans the working tree *and every reachable Git
revision*.** A banned literal that reaches a commit does not become clean when a later commit
removes it — the scan still finds it in history, and the repository rules forbid rewriting
published history. A single careless paste therefore permanently red-lights the gate.

The literals it rejects, verified in the scan source: the source customer identifier, the source
API/domain identifier, the source account identifier, the proof-of-concept identifier, the source
architecture acronym, Grafana Cloud and service-account token prefixes, private Tailscale
hostnames, the local macOS home-directory path prefix, private-key PEM headers, AWS ARNs carrying
an account ID, JWT-shaped values, `kind: Secret` documents, private-range IPv4 HTTP endpoints, the
forbidden filenames (`terraform.tfvars`, `.env`, `.envrc`, `config.json`), and tracked archive, key
container or database file extensions.

**The trap that catches agents specifically: absolute local paths.** Tooling instructions, hook
tests, scratch notes and pasted command lines carry them by default, and the scan rejects the home
path prefix case-sensitively. Anything committed here derives its paths — from `git rev-parse
--show-toplevel`, from `CLAUDE_PROJECT_DIR`, or relatively. Never hard-code one. The reference hook
test that this repository's guard was copied from hard-coded them, and would have failed this
repository's own gate unchanged.

## The gate

```bash
./scripts/validate.sh
```

One command, and it is the whole local gate: the public-release scan, `gofmt`, `go mod tidy` with a
`git diff --exit-code` on `go.mod`/`go.sum`, race-enabled tests with coverage, `go vet`, a YAML
parse of every tracked YAML document, four Kustomize renders, the example-README coverage check,
and the ApplicationSet watch-path assertion. `definition_of_done` in `backlog/config.yml` carries it
plus the hosted run, so every task inherits both.

**A green local gate is not sufficient for `Done`.** The hosted `Validate public reference` workflow
must pass on the completing commit, and its run ID belongs in the task's final summary alongside the
SHA — that is the convention the closed issue history already established and it is worth keeping.

## Recurring defects in this codebase

These have each cost a real debugging cycle. Check them before writing composition code, not after.

**Deterministic children need an explicit external name.** Every resource whose import identity is
derivable from the request renders `crossplane.io/external-name` — stacks by slug, folders as
`<uid>-folder`, dashboards by uid, roles and whole-role assignments by role uid, provider options by
provider name. Omit it and a non-destructive orphan-and-adopt transition tries to *create* a
resource that already exists, which is precisely the failure mode the non-destructive management
policies exist to prevent.

**Provider-assigned IDs cannot be derived, only observed or inventoried.** StackServiceAccount,
AccessPolicy and Team identities are assigned by the provider. Guessing one is always wrong.
`renderTeamAccess` reads `status.atProvider.teamId` from the observed resource and **emits nothing
at all until it is present** (`platform/function/roles.go:99`, gated at :125 and :146) — that
deliberate wait is the pattern to copy, not a missing case to fill in.

**Role assignment items are three segments, not four.** The external name is
`roleUID + ":team:" + teamID` using the bare observed team ID. Passing the Team *reference* through
instead yields an org-qualified value, and the provider then builds an invalid four-segment ID. Both
call sites are in `platform/function/roles.go`; the contract is pinned by tests.

**The pinned provider's Role initializer errors when `autoIncrementVersion` is omitted**, so Roles
render it explicitly as `false` while continuing to omit the deprecated server-managed `version`
field. This is a workaround against a specific pinned provider version — if the pin moves, re-check
it rather than assuming it is still needed.

**Every catalog directory must be a renderable Kustomize base with a README.** A consumer applying
Kustomize patches forces Kustomize rendering for *every* selected catalog path, so a directory
without a `kustomization.yaml` fails before deployment even though it validates in isolation. This
shipped once: `examples/catalog/minimal` lacked one and broke a consumer render. The gate now checks
README coverage, but it does not check that every catalog directory renders — only `comprehensive`
and `minimal` are rendered explicitly. **Adding a catalog directory means adding its render to
`scripts/validate.sh` too.**

**The ApplicationSet must watch only top-level `enabled/*`.** `scripts/validate.sh` asserts the
generator's `directories` equals exactly `[{path: enabled/*}]`. `enabled/` starts empty and inert;
`examples/` is never watched. Widening that glob is how inert examples become live requests.

**Whole-set resources need exactly one declarative owner.** Folder and dashboard ACLs and role
assignments replace the entire set. Two owners silently fight. Any change that adds a second writer
to one of these is a design error, not a merge conflict to resolve.

## Lane conventions

Natural boundaries, each with a single owner per wave:

- `platform/function/*.go` — the composition function. `fn.go`, `roles.go`, `access.go`,
  `options.go` and `plugins.go` split cleanly by feature area, but `fn_test.go` is **one file with
  one owner**, and it is where lanes collide first. A wave touching two feature areas gives the test
  file to one lane or defers it to the wiring pass.
- `platform/apis/`, `platform/rbac/`, `platform/provider/` — the declarative surface.
- `examples/catalog/*` — one lane may own several directories; they do not interact.
- `deploy/` — installation and GitOps integration.
- `docs/` and `README.md` — the README is a single 54 KB file and therefore a single owner, always.
- `scripts/validate.sh` — an integration file. Never edited by two lanes in parallel; changes to it
  belong to the wiring pass, because it is what every other lane is being judged by.

**The escape hatch:** a lane that needs a change inside another lane's file stops and returns the
exact edit as a blocker rather than making it. The wiring pass applies it. A lane that finds the
gate already red on `main` before it starts also stops — it is not that lane's failure to fix.

## The exclusive resource: function package publishing

Publishing the composition function is **serialized, single-owner, and cannot run in parallel with
anything that depends on it.** The sequence is fixed: land the code change, let the publish workflow
build and sign a multi-platform OCI index, verify the signature against the exact main-branch
publish workflow identity, then pin the resulting immutable digest in `platform/function/install.yaml`
in a follow-up commit. The digest appears **twice** in that file — the Cosign verification Job's
args and the `Function` package reference — plus once in the Job's name suffix. All three move
together.

Consequences worth stating because they have bitten: the digest cannot be known before the workflow
runs, so a wave cannot pre-write it; the intermediate commit legitimately carries the *previous*
digest; and a function behaviour change is not actually delivered until the pin moves, even though
the gate is green.

## Run-end against this tracker

Task state is the record, so the run's closing terminal message is a covering note only — what this
run learned that no single task captures. Nothing durable may live only there.

- landed work → `Done`, with the completing SHA **and the hosted validation run ID** in the final
  summary, finalized in one call (`--check-ac ... -s Done`);
- attempted and blocked → `Parked`, with a concrete resume boundary. Blocked on the publish workflow
  or on a digest that does not exist yet is the common and legitimate case here;
- discovered work → a new task labelled `needs-triage`;
- untouched work is self-evidently still `To Do`.

Writing the report is the last unit of work, not a reply to a request.
