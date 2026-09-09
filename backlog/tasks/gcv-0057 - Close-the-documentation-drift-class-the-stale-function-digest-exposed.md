---
id: GCV-0057
title: Close the documentation-drift class the stale function digest exposed
status: To Do
assignee: []
created_date: '2026-09-09 08:10'
labels: []
dependencies: []
ordinal: 57000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The wave 6 reviewer found `docs/installation.md` quoting a function package digest two publishes stale, in the very table row that names `platform/function/install.yaml` as canonical for it. GCV-0042 had just reconciled that file and reported it correct. Nothing in the gate compared a documented value against its source, so a publish that moved the pin left the doc silently wrong and every check stayed green.

Commit 7cbf10e fixed that one digest and added a gate assertion for digests specifically. The class is wider: the same pinned-versions table quotes Crossplane, ESO, cosign and function SDK versions that live in `deploy/`, `platform/` and `go.mod`; the catalog and request-schema references enumerate kinds and directories that live under `platform/apis/` and `examples/catalog/`. Each is a hand-maintained restatement of a machine-readable fact, and each can go stale the same way. GCV-0036 closed this class for the XRD/renderer registry; this task closes it for documentation.

Prefer discovery over a hand-maintained list of what to check, matching how the existing manifest checks in `scripts/validate.sh` are written.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every version and identifier the pinned-versions table quotes is checked against the file that actually carries it, and disagreement fails the gate by path
- [ ] #2 Documentation that enumerates shipped kinds or catalog directories is checked against what the repository actually ships, so a new surface cannot land undocumented
- [ ] #3 Each new check is proven by a weaken/reject/restore control, not by observing that the gate is currently green
- [ ] #4 The checks discover their inputs rather than hard-coding filenames, so a new document or manifest is covered without editing the gate
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
