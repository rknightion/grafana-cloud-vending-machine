---
id: GCV-0102
title: >-
  Record and document what happens to a vended connection Secret when its
  managed resource is removed
status: In Progress
assignee: []
created_date: '2026-10-04 20:18'
updated_date: '2026-10-04 20:21'
labels:
  - docs
  - security
dependencies: []
priority: medium
type: bug
ordinal: 102000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
During GCV-0078 (migrate the pre-organization estate onto the current request schema) a security review found that a Retain or no-Delete policy on an external token does not stop Kubernetes garbage collection deleting the token's connection Secret when the managed resource itself is removed, because the Secret is owned by the managed resource. A reader of the repository's secrets, decommission and controlled-deletion documentation could conclude that retaining the external object also retains the credential in-cluster. Nothing has yet checked what the renderer and the documentation actually promise about connection Secret lifetime across managed-resource removal, decommission and the platform-authorized Delete lifecycle.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every documentation statement about connection Secret or credential retention across managed-resource removal, decommission and the controlled Delete lifecycle is inventoried with file and line, and each is classified as accurate or misleading against pinned Crossplane runtime source at an exact revision
- [ ] #2 Misleading statements are corrected so the docs state that external-object retention does not retain the in-cluster connection Secret, and say what an operator must prove about Secret retention before authorizing any managed-resource removal
- [ ] #3 If the renderer emits anything that contradicts the corrected docs, the lane stops and returns the question with evidence instead of changing renderer behaviour; otherwise no renderer change is made
- [ ] #4 No estate identity appears in the change, and just check passes
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Inventory retention claims against exact pinned runtime source, correct misleading documentation only, and stop on a renderer contradiction; validate offline.
<!-- SECTION:PLAN:END -->
