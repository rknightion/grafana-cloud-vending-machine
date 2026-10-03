---
id: GCV-0100
title: Rebase the provider carry onto the current upstream release
status: In Progress
assignee: []
created_date: '2026-10-02 16:48'
updated_date: '2026-10-03 13:17'
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

Loop25 carry attempt C1-impl-2 admitted after a fresh audited clean fork worktree replay. Prior runtime API compile failure is the changed premise; explicit config carry cases and documented tests must pass before the single tag publication. Independent review and exact-SHA local plus hosted validation remain required. C1-impl-1 remains consumed; impl-2 and impl-3 are the granted successor ladder.

Loop25 C1-impl-2 consumed on a guessed API package path that still failed compilation, before publication. Final granted C1-impl-3 inspected installed API and generated types, ported only test import and policy alias, preserved all assertions, and passed all thirteen carry cases plus documented tests at 4bf00dd8901590aef8770314dd1393e2041474da. Two setup/toolchain infrastructure retries; no tag push yet. Independent final review admitted. GCV pin candidate correctly rejects its explicitly root-owned digest placeholder; signatures, tagged CI, full local validation and hosted completion remain pending. No further implementation attempt.

Loop25 final independent C2 review ACCEPTS exact fork 4bf00dd8901590aef8770314dd1393e2041474da and pin candidate; all ninety-two activated-kind map entries match served source CRDs. Root published the single accepted carry tag; hosted tag CI37124509433 passed all seven jobs on that SHA. Root Cosign3.1.3 verified upstream package and carried multi-platform index against exact tag-scoped workflow identities. Both provider runtime/verifier placeholders now use the identical verified index digest. Local combined gate and main hosted validation remain pending; no completion claim yet.
<!-- SECTION:NOTES:END -->
