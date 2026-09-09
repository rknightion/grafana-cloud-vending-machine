---
id: GCV-0060
title: Vend the asserts product surface
status: To Do
assignee: []
created_date: '2026-09-09 08:10'
labels: []
dependencies: []
ordinal: 60000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The wave 6 coverage measurement diffed all 297 CRDs in the pinned provider v2.14.0 against the kinds this function emits. `asserts` was the largest group left deliberately uncommissioned: 2 of 18 kinds vended, missing thresholds, log/trace/profile configs, custom model rules and prom rule files. Wave 6 recorded the reason rather than the omission - it is a coherent product surface that deserves its own design pass, not a lane bolted onto the end of another wave.

This task is deliberately NOT owned by the wave 7 consolidation campaign. It is expected to run as its own concurrent effort. The ownership boundary that keeps the two from colliding: this work owns new `platform/apis/asserts-*.yaml`, new `platform/function/asserts*.go` and new `examples/catalog/asserts*/` files only. `platform/kustomization.yaml`, `platform/function/fn.go`, `platform/provider/managed-kind-map.json`, `scripts/validate.sh`, `docs/` and `platform/function/install.yaml` are shared integration surfaces owned by whichever campaign is designated the integrator; a second writer on any of them is a design error, not a merge conflict to resolve.

The publishing sequence is an exclusive resource. Two campaigns pushing to `main` will race on the function publish and on release-please. Only one may hold it at a time.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The asserts surface is vended from platform-controlled input with an explicit external name on every deterministic child
- [ ] #2 A platform-owned bound caps whatever the surface can multiply, and an over-budget request is refused with its own message, proven against the real API server
- [ ] #3 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [ ] #4 A catalog example renders inert and is covered by the catalog README and Kustomization checks
- [ ] #5 No file outside the ownership boundary in this description is edited by this work without an explicit integrator handoff
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
