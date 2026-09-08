---
id: GCV-0043
title: Reconcile the function package after concurrent runtime dependency automation
status: To Do
assignee: []
created_date: '2026-09-08 21:34'
labels:
  - needs-triage
dependencies: []
type: bug
ordinal: 43000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A concurrent runtime dependency update landed while wave 5 was in flight. The publisher's runtime-change detector ran from the function subdirectory and incorrectly reported that the change did not affect runtime inputs, so no package was published and the immutable package references were not moved. Reconcile the retained source and package state before release.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decide whether the concurrent dependency bump remains the intended function source.
- [ ] #2 If retained, publish and sign from the exact source SHA and verify workflow identity.
- [ ] #3 Move all function package digest references together to the verified immutable digest.
- [ ] #4 Prove just check and exact-SHA hosted validation.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
