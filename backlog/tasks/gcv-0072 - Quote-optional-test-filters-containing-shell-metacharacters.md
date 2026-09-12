---
id: GCV-0072
title: Quote optional test filters containing shell metacharacters
status: Done
assignee: []
created_date: '2026-09-12 17:07'
updated_date: '2026-09-12 19:00'
labels:
  - needs-triage
dependencies: []
type: bug
ordinal: 72000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The top-level test recipe interpolates its optional filter directly into a shell command. A regex alternation passed as the filter is parsed as a pipeline, so the first test process runs and later alternates are executed as commands. Wave 11 encountered this while selecting several focused tests; the whole repository gate is unaffected.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A filter containing regex alternation is passed to go test as one argument and selects the intended tests
- [x] #2 A simple filter and an omitted filter retain their current behavior
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Fixed by shell-quoting the optional filter in the top-level `test` recipe: `{{ if filter != "" { "-run " + quote(filter) } else { "" } }}`. just 1.58.0's `quote()` emits a single shell-quoted word, so an alternation reaches go test as one argument instead of being parsed as a pipeline.

AC1 proven: `just test 'TestProviderConfigIsNotReadyBeforeObservation|TestObservedProviderConfigDoesNotMaskUnreadyChild'` expands to `-run 'TestProviderConfigIsNotReadyBeforeObservation|TestObservedProviderConfigDoesNotMaskUnreadyChild'` and reports 8.9% coverage, against 8.4% for the single-test form, so both alternates were selected rather than the first running and the second executing as a command.

AC2 proven: the simple-filter form is byte-identical apart from the added quotes, and the omitted-filter form still expands to `go test -race -cover  ./...` with no `-run`.

Validated as declarative task-surface config rather than with a new unit test: `just --dump --dump-format json` exits 0 and `just --fmt --check` reports no change. No CodeRabbit review: a recipe quoting fix is task-surface config with no branching logic.

Gate: local `just check` passed at 85.0% coverage with "Validation passed."
<!-- SECTION:NOTES:END -->
