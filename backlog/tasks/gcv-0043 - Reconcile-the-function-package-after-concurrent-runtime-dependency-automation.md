---
id: GCV-0043
title: Reconcile the function package after concurrent runtime dependency automation
status: To Do
assignee: []
created_date: '2026-09-08 21:34'
updated_date: '2026-09-08 21:59'
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

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closeout update: concurrent security dependency automation merged as source SHA 629c39c2b42e6294df0bdc3c59442846940779f0. The corrected runtime-change detector selected that source; publisher run 34282605376 completed its test, vet, amd64 build, arm64 build, multi-platform push, and signature steps for immutable tag v0.0.0-629c39c2b42e. The checked-in function package references still point to the previous digest. Resume by resolving the new tag to its immutable digest, verifying its signature and source/run identity, then moving both package references together. Do not republish unless that verification fails.
<!-- SECTION:NOTES:END -->
