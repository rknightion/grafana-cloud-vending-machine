---
id: GCV-0072
title: Quote optional test filters containing shell metacharacters
status: To Do
assignee: []
created_date: '2026-09-12 17:07'
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
- [ ] #1 A filter containing regex alternation is passed to go test as one argument and selects the intended tests
- [ ] #2 A simple filter and an omitted filter retain their current behavior
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
