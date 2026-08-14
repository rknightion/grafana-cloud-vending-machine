---
id: doc-0003
title: Closed GitHub issues — pre-Backlog history index
type: other
created_date: '2026-08-14 16:39'
updated_date: '2026-08-14 16:43'
---
All work before 2026-08-14 was tracked as GitHub Issues. Three issues existed, all closed as
completed. This is the index of them.

**The issues were archived and then deleted from GitHub on 2026-08-14, so `gh issue view <N>` no
longer resolves any of them and this index is a pointer to the archive, not to GitHub.** The full
verbatim text — every body and all 11 comments — is committed at `archive/issues-dump.json`, with
`archive/README.md` describing the capture and the redaction finding. Nothing was redacted; the
archive is verbatim. It was committed and pushed before anything was deleted, and every deleted
number was asserted present in it first.

The Issues tab stays enabled for external contributors; it is simply empty of this project's own
history.

```bash
jq -r '.[] | select(.number == 4) | .body, "\n--- comments ---", (.comments[] | "\n[\(.createdAt)]\n\(.body)")' \
  archive/issues-dump.json
```

They were deliberately **not** imported as `Done` tasks. Backlog IDs follow creation order, so a
`GCV-000N` could never be made to match the `#N` already cited in this repository's commit
trailers — that would create a second ID space over the same history. The original `#N` numbering
stays the only one, and the board carries only what is left to do.

## The three issues

| # | Title | Closed | Closed by |
|---|---|---|---|
| 1 | Build portable public Grafana Cloud vending-machine reference | 2026-08-04 | `3fd7a04` (`Closes #1`) |
| 3 | Add an executable comprehensive reference and final public-release audit | 2026-08-04 | `7f57a56` (`Closes #3`) |
| 4 | Harden child adoption and clarify example enablement | 2026-08-05 | `63e2eab` (`Closes #4`) |

`#2` was a pull request, not an issue, and is not in the archive.

**Issue 1 — the build.** Established the whole repository: the namespaced `GrafanaCloudStackRequest`
and `GrafanaCustomRoleBinding` APIs, pipeline Compositions backed by a repository-owned Go
composition function, rotating administrator and telemetry credential chains, baseline content and
reconciliation modes, the optional SSO / report / plugin / incident / team / custom-role surface, and
the Crossplane, External Secrets, AWS workload-identity and Argo CD deployment examples. It also set
the binding decisions that still hold: `platform.example.org` as an explicit placeholder API group,
non-destructive deletion defaults, and never copying live stack claims or real secret material.
Delivered across `c5b0453`, `b51fe2f`, `7c3a37b` and `3fd7a04`. Its long tail was a GitHub-side
blocker rather than a code one — personal GHCR package visibility is a web-only, irreversible
setting, so the package had to be made public by hand before an anonymous pull could be proven.

**Issue 3 — the comprehensive example and the publication audit.** Consolidated every public API kind
into one renderable `examples/catalog/comprehensive` Kustomize base, documented the
public-base/private-overlay consumption pattern, and audited tree, history, workflows, OCI
references and generated artifacts for publication risk. Closed once in `3b3e4b0`, then **reopened**
when the first external consumer hit a real defect: applying Kustomize patches forces Kustomize
rendering for *every* selected catalog path, and `examples/catalog/minimal` had no
`kustomization.yaml`, so it failed before deployment despite validating in isolation. Fixed and
re-closed in `7f57a56`. That defect is carried forward in the Wave operating model document,
because the gate still only renders two catalog directories explicitly.

**Issue 4 — provider-contract hardening.** Closed the gaps the first external GitOps consumer found.
Deterministic children now render `crossplane.io/external-name` so a non-destructive
orphan-and-adopt transition adopts instead of attempting creation; Roles render the pinned
provider's `autoIncrementVersion: false` initializer workaround while omitting the deprecated
server-managed `version` field; RoleAssignmentItem identities wait for the observed bare team ID and
emit the documented three-segment `roleUID:team:teamID` form rather than an org-qualified reference
that would produce an invalid four-segment ID. It also moved the opt-in live boundary from
`examples/enabled` to top-level `enabled/` and gave every catalog directory a README. Delivered
across `c478b9f`, `fcd3bc5` and `63e2eab`. Each of those provider contracts is now pinned by tests
and restated as a recurring defect in the Wave operating model document.

## What the closed set is worth keeping for

Two things, both already carried forward rather than left here: the provider contracts above, which
are easy to reintroduce and expensive to debug, and the operating conventions the issue comments
established by practice — a completing commit carries `Closes #<issue>`, and a completion claim
carries both the SHA and the hosted validation run ID as evidence. The tracker keeps the second one
as `definition_of_done`.
