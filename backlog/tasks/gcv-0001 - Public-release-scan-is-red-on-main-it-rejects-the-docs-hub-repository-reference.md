---
id: GCV-0001
title: >-
  Public-release scan is red on main: it rejects the docs-hub repository
  reference
status: Done
assignee: []
created_date: '2026-08-14 16:41'
updated_date: '2026-08-18 10:18'
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
- [x] #1 `./scripts/validate.sh` exits 0 on a clean checkout of main, with the exit code shown as evidence rather than the tail of its output
- [x] #2 The scan still fails when the identifier appears in a genuine leak context, proven by a deliberate negative test that is then reverted
- [x] #3 The narrowing is scoped to the specific legitimate references and does not blanket-exclude the pattern or the files that contain it
- [x] #4 The reason the control was narrowed is recorded in the scan source next to the exclusion, so a future reader does not restore the false positive
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Fixed in df56993. Hosted `Validate public reference` run 32125954492 on df56993: success. Local `./scripts/validate.sh` exit 0.

**The narrowing.** A new `scan_fixed_allowing` handles the one identifier that names both a source environment and the public documentation hub this repository publishes into. It does not exclude the pattern or the files. Each raw hit record has its allowed substrings removed and the remainder is re-tested against the pattern; only records where nothing forbidden survives pass, and the original line is what gets printed on failure. The rationale sits in a comment directly above the call, and the allowlist is built from the same split-string variable as the pattern so the control still does not reproduce the identifier in source.

Negative tests run against the working tree, each reverted (AC2):

- a private endpoint form of the identifier -> exit 1
- a line carrying BOTH an allowed hub reference and environment identity -> exit 1, so a mixed line cannot launder a leak
- an allowed substring present only in the FILE PATH, with environment identity in the content -> exit 1, so a path match does not exonerate a line
- an allowed substring in the path with clean content -> exit 0

The first three were the failure modes of a naive whole-record `grep -v` allowlist, which was the first attempt here and was rejected in review before it was committed.

**Second defect found, not in the original report.** ripgrep is absent from the ubuntu-24.04 hosted runner. Every `rg` invocation was failing with `command not found`, so on CI the working-tree half of EVERY check and both archive/binary checks were silent no-ops - the hosted scan had been covering reachable history alone, and would have reported a false pass on a working-tree-only leak once the history hit was cleared. Two changes: the workflow installs ripgrep before running the gate, and the script now exits 1 up front if `rg` is missing rather than degrading silently.

Also hardened while in there: search invocations go through a `run_search` wrapper that treats status 1 as no-match and any status above 1 as a hard failure, so a broken search can no longer be swallowed as a clean result. This applies to the new code path only; `scan_fixed`, `scan_fixed_case_sensitive` and `scan_regex` still use the bare `if <search>` form and cannot distinguish no-match from error. That is pre-existing and was left alone as out of scope - worth a follow-up if anyone wants the whole file consistent.

CodeRabbit review was run before the commit and raised both the whole-record allowlist flaw and the error-swallowing; both were fixed before anything was pushed.
<!-- SECTION:NOTES:END -->
