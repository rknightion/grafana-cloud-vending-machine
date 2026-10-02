---
id: GCV-0100
title: Rebase the provider carry onto the current upstream release
status: In Progress
assignee: []
created_date: '2026-10-02 16:48'
updated_date: '2026-10-02 18:55'
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

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Reserve admission after the signed function pin is green: rebase carried rotation and GHCR-only publication commits onto upstream v2.15.0; prove all three rotating-token kinds; review fork code; publish exactly one branch/tag to the carry remote; verify tag CI and both package signatures; prepare provider manifest, installation rows and required map digest in an isolated candidate; independent C2 review and root main gate/CI before release.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Loop23 reserve admitted after P5 SHA 91ae4568f7866d00e294aff843b92676d701c1b1 passed local gate and hosted Validate 37049584845. Exactly C1-impl-1 is reserved, no second implementation or review-repair in this reserve. Root corrected the packet ownership gap for the required providerPackageDigest metadata field; all other managed-kind-map contents and released APIs remain frozen. P8 starts without the reserve on its failure/rejection or the frozen cutoff.
<!-- SECTION:NOTES:END -->
