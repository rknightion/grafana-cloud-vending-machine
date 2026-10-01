---
id: GCV-0098
title: Distinguish the default-branch HEAD alias from ungranted loop audit writes
status: To Do
assignee: []
created_date: '2026-10-01 00:29'
labels:
  - needs-triage
dependencies: []
priority: low
type: bug
ordinal: 98000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Loop 22 closeout reports the remote symbolic HEAD movement as ungranted while the identical refs/heads/main movement is granted. This is distinct from the correctly flagged release-branch non-fast-forward rewrite and foreign automation pull refs. The extra alias warning obscures attribution in otherwise authorized campaigns. The maintained audit implementation and canonical grant procedure live outside this repository; do not edit generated runtime copies here.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A granted default-branch movement does not independently fail solely because its remote HEAD alias moves to the same commit.
- [ ] #2 Ungrantable independent branch changes and non-fast-forward changes remain blocked, with complete raw evidence retained.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
