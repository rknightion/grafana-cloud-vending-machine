---
id: GCV-0001
title: >-
  Public-release scan is red on main: it rejects the docs-hub repository
  reference
status: To Do
assignee: []
created_date: '2026-08-14 16:41'
updated_date: '2026-08-14 16:42'
labels:
  - needs-triage
dependencies: []
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
`./scripts/validate.sh` fails at its first stage on a clean checkout of main, and has done since a9ab075 (2026-08-10), the commit that added the documentation-site wiring. Verified 2026-08-14 by stashing all local work and running the gate on an untouched tree: exit 1.

The scan bans the source API/domain identifier as a fixed string, and the docs-site wiring legitimately names the documentation hub repository and its domain in four places: two comments in .gitignore, two comments in docs.toml, and the repository input in .github/workflows/trigger-docs-sync.yml. These are references to a sibling public repository, not source-environment identity leaks, so the control is firing on a false positive rather than catching a real one.

It fails in BOTH halves of the scan - working tree and reachable Git history - and the history half is the hard part. A commit that removes the strings from the working tree does not clear the history match, and rewriting published history is not an option for this repository. So the fix has to be a narrowing of the control itself, not a content change.

Everything else in the gate is green as of 2026-08-14: gofmt clean, go mod tidy clean, race tests pass at 89.0 percent coverage, go vet clean, YAML parse clean, all four Kustomize renders succeed.

Nothing was changed unilaterally here - narrowing a publication safety control is a judgement call about what the control is for, so it is filed rather than fixed.

Note for whoever picks this up: do not quote the offending strings into this task, a commit message body, or any tracked file while investigating. The first draft of this description quoted the triggering commit subject verbatim and would itself have added a fresh, permanent occurrence to history. Refer to commits by SHA.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `./scripts/validate.sh` exits 0 on a clean checkout of main, with the exit code shown as evidence rather than the tail of its output
- [ ] #2 The scan still fails when the identifier appears in a genuine leak context, proven by a deliberate negative test that is then reverted
- [ ] #3 The narrowing is scoped to the specific legitimate references and does not blanket-exclude the pattern or the files that contain it
- [ ] #4 The reason the control was narrowed is recorded in the scan source next to the exclusion, so a future reader does not restore the false positive
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
