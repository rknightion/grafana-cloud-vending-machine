---
id: GCV-0100
title: Rebase the provider carry onto the current upstream release
status: Parked
assignee: []
created_date: '2026-10-02 16:48'
updated_date: '2026-10-02 21:56'
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

Loop23 reserve C1-impl-1 consumed 1/1, infrastructure retries 0, review-repair 0 allowed, no ceiling extension. Conflict-free uncommitted replay onto upstream v2.15.0 passed documented make test, but that target excludes config tests. Explicit carry tests failed before execution because config/rotation_test.go:18 imports crossplane-runtime/v2/apis/common/v1, absent from the target runtime v2.4.0 API. No rotation cases ran; no commit, branch/tag push, signed artifact, GCV candidate or C2 review. Current accepted v2.14.0-rotation.1 stays pinned. Preserve staged six-file fork diff on the local reserve branch; diagnostic patch SHA-256 331dd7ff1b593874900ae06dfe8b3d892c2be7966487d23ed53b1a14c1af893d. Resume only with new attempt authority: port the obsolete common API test import, then exercise all three kinds and resolve any further compatibility failures before publication. CodeRabbit completed all six files; one minor missing ready_for_rotation guard finding remains on the unshipped candidate and must be reconsidered in the successor.

Loop24 provider carry admission was blocked before dispatch by the required clean-checkout preflight: six staged replay paths retained from the prior attempt. No implementation attempt consumed, no commit or branch/tag push, no GCV candidate. Preserve the staged diagnostic replay; resume only after the owner reconciles the checkout and the required preflight passes. Granted implementation attempts remain unused; current released carry remains unchanged.
<!-- SECTION:NOTES:END -->
