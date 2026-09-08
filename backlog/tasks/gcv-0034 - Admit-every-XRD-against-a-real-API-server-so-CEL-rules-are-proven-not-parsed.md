---
id: GCV-0034
title: 'Admit every XRD against a real API server so CEL rules are proven, not parsed'
status: To Do
assignee: []
created_date: '2026-09-08 17:02'
labels: []
dependencies: []
ordinal: 34000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every wave-3 lane closed with the same unproven line: the XRD CEL validation rules were parsed and rendered locally, never admitted by a Kubernetes API server. Thirteen XRDs now carry structural schemas, x-kubernetes-validations rules, immutability rules and preserve-unknown-fields subtrees whose real behaviour under apiserver CEL, pruning and ratcheting has never been observed. A locally parsed rule that the apiserver rejects at install time, or silently prunes before evaluating, is indistinguishable from a working one under the current gate. An ephemeral local control plane is not a live Grafana Cloud stack and does not breach the no-live-environment constraint.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every XRD in platform/apis/ is installed into an ephemeral local Kubernetes API server and accepted without error
- [ ] #2 Each XRD's structural schema is confirmed accepted by the apiserver, including every x-kubernetes-validations rule, so a CEL expression that fails apiserver compilation fails the harness
- [ ] #3 A valid example request for each API is admitted through the real apiserver, not merely parsed
- [ ] #4 The harness is reproducible from a clean checkout with one just recipe and pins its control-plane version explicitly
- [ ] #5 The recipe is placed in check or ci according to the frozen task-surface rule, with a comment naming which heavy dependency it needs if it lands in ci
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
