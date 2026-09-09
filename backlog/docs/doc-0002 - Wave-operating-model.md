---
id: doc-0002
title: Wave operating model
type: guide
created_date: '2026-08-14 16:36'
updated_date: '2026-09-09 00:17'
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

## Grafana Cloud only, deliberately

**This vending machine supports Grafana Cloud stacks and will not support self-managed Grafana.**
Decided by the repository owner 2026-09-08. The consequence worth stating, because it removes a
caveat that otherwise gets copied forward: preconditions that apply only to self-managed
deployments are not constraints here. Grafana feature toggles are the live case — no provider
resource sets a feature toggle on a Cloud stack, and several surveyed surfaces list two
self-managed toggles as a precondition. Those rows are irrelevant to this repository and must not be
recorded as blockers. Preview status is a separate question and remains a real caveat.

## The gate

```bash
just check
```

One command, and it is the whole local gate: it runs `scripts/validate.sh`, byte for byte the same
script the hosted `Validate public reference` workflow runs, covering the public-release scan,
`gofmt`, `go mod tidy` with a `git diff --exit-code` on `go.mod`/`go.sum`, race-enabled tests with
coverage, `go vet`, a YAML parse of every tracked YAML document, platform and environment Kustomize
renders plus every discovered catalog base, recursive platform-manifest coverage, XRD/renderer
registry agreement, emitted-kind activation coverage against the pinned provider CRD map, discovered
signed-package/verifier digest agreement, and the exact ApplicationSet watch-path assertion. The
admission harness installs every XRD at the pinned real API server, admits catalog examples, and
proves fail-closed rules with create/update and weaken/admit/restore controls. Explicit SCIM null
is admitted and persisted with its key present, then refused by the renderer with zero children;
that reconcile-time boundary was accepted by the owner on 2026-09-08. `definition_of_done` in `backlog/config.yml` carries it plus
the hosted run, so every task inherits both.

Discover the task surface rather than guessing it: `just --list`, `just --dump --dump-format json`,
`just --show <recipe>`. Prefer `just <recipe>` over the underlying tool.

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
README and Kustomization coverage, discovers every catalog directory, checks all sibling manifest
entries, and renders every directory. **Adding a catalog directory requires no hand-kept render-list
entry.** A missing directory manifest or resource entry must fail the gate by path.

**The ApplicationSet must watch only top-level `enabled/*`.** `scripts/validate.sh` asserts the
generator's `directories` equals exactly `[{path: enabled/*}]`. `enabled/` starts empty and inert;
`examples/` is never watched. Widening that glob is how inert examples become live requests.

**Whole-set resources need exactly one declarative owner.** Folder and dashboard ACLs and role
assignments replace the entire set. Two owners silently fight. Any change that adds a second writer
to one of these is a design error, not a merge conflict to resolve.

## Lane conventions

Natural boundaries, each with a single owner per wave:

- `platform/function/*.go` — the composition function. **One feature area is one file, and a new
  feature area gets a new file plus its own `_test.go`.** Decided by the repository owner
  2026-09-08, replacing the earlier convention that piled new tests into `fn_test.go`: that made
  `fn_test.go` the file every lane collided in first, which capped a wave at roughly two feature
  areas regardless of how much independent work existed. `renderStack` in `fn.go` is reduced to a
  frozen call list of `add<Feature>(...)` registration functions, one per feature file, and the call
  list belongs to the root or the wiring pass — never to a lane.
- `platform/apis/` — **one API is one file**, named for the API rather than only its version, and a
  new API arrives as a new file. `platform/kustomization.yaml`, the
  `ManagedResourceActivationPolicy` kind list and `platform/rbac/composition-rbac.yaml` are
  append-only registries that cannot be split into per-lane files, so their entries are pre-assigned
  in the goal and applied by the wiring pass. A lane wanting an entry other than its assigned one
  stops and says so.
- `platform/provider/` — the provider pin and its signature verification. The digest appears three
  times in one file and always moves together.
- `examples/catalog/*` — one lane may own several directories; they do not interact.
- `deploy/` — installation and GitOps integration.
- `docs/` — one subject per page, following the `docs.toml` navigation. Lanes own their subject
  pages; the wiring pass owns navigation and shared manifest updates. `README.md` is first-minute
  orientation and a link map only. The former single-owner README exception and its scheduling
  dependency are retired: a subject lane can deliver its documentation with its implementation.
- `scripts/validate.sh` — an integration file. Never edited by two lanes in parallel; changes to it
  belong to the wiring pass, because it is what every other lane is being judged by.

**The escape hatch:** a lane that needs a change inside another lane's file stops and returns the
exact edit as a blocker rather than making it. The wiring pass applies it. A lane that finds the
gate already red on `main` before it starts also stops — it is not that lane's failure to fix.

**Lane-local validation, because the registries are wiring-pass owned.** A lane cannot run `just
check` meaningfully before its kustomization and activation-policy entries exist, so a lane's own
acceptance check is its package tests plus a YAML parse of the documents it wrote. One named gate
owner runs `just check` after the wiring pass, against the integrated tree.

## The exclusive resource: function package publishing

Publishing the composition function is **serialized, single-owner, and cannot run in parallel with
anything that depends on it.** The sequence is fixed: land the code change, let the publish workflow
build and sign a multi-platform OCI index, verify the signature against the exact main-branch
publish workflow identity, then pin the resulting immutable digest in `platform/function/install.yaml`
in a follow-up commit. The digest appears **twice** in that file — the Cosign verification Job's
args and the `Function` package reference. Both move together. The verification Job uses a stable
PreSync-hook name with BeforeHookCreation; it has no digest suffix to update.

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
