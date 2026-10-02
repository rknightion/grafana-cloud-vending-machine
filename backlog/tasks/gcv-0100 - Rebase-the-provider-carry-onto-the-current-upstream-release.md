---
id: GCV-0100
title: Rebase the provider carry onto the current upstream release
status: To Do
assignee: []
created_date: '2026-10-02 16:48'
labels: []
dependencies: []
priority: medium
type: chore
ordinal: 100000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The carried rotation controller is based on the previous upstream package. Evaluate and ship a rebased carry without changing in-place token behavior.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 All three rotating-token kinds preserve in-place rotation on the current upstream release
- [ ] #2 Tagged build and upstream package signatures verify and independent review accepts
- [ ] #3 Provider pins land with local gate and hosted validation evidence
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
