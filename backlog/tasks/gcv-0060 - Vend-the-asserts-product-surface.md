---
id: GCV-0060
title: Vend the asserts product surface
status: To Do
assignee: []
created_date: '2026-09-09 08:10'
updated_date: '2026-09-09 12:46'
labels: []
dependencies: []
ordinal: 60000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The wave 6 coverage measurement diffed all 297 CRDs in the pinned provider v2.14.0 against the kinds this function emits. `asserts` was the largest family left deliberately uncommissioned - thresholds, log/trace/profile configs, custom model rules and prom rule files are all unvended. Wave 6 recorded the reason rather than the omission: it is a coherent product surface that deserves its own design pass, not a lane bolted onto the end of another wave.

**Correction.** This description previously said "2 of 18 kinds vended". That was WRONG. **Zero** asserts kinds are vended: `platform/provider/managed-kind-map.json` carries 80 managed kinds and none is in the asserts family, and there is no `platform/apis/asserts-*.yaml`, no `platform/function/asserts*.go` and no `examples/catalog/asserts*/`. The family's true kind count is also not derivable from anything committed here - `platform/function/testdata/provider-crds.json` caches only the 30 CRDs the tests need - so it must be derived from the exact pinned provider artifact, by the method GCV-0036 used to verify against all 297.

`docs/architecture.md` assigns the family this treatment: "Use a separate opt-in onboarding module because entitlement and additional metrics/Grafana credentials are required." Vending it as an opt-in module satisfies that line rather than contradicting it, and the line must be updated to name the module once it exists. `docs/faq.md` separately lists Asserts alongside ML as "separately owned"; that sentence is already stale on ML, which wave 6 vended.

Entitlement cannot be proven in this repository, which makes no live Grafana Cloud, cluster or source-environment contact. The surface is proven to the same standard as every other module: admission against a real API server, deterministic render with explicit external names, and an inert catalog example.

Integration surfaces this work must reach and therefore owns as the sole campaign in flight: `platform/kustomization.yaml`, `platform/function/fn.go`, `platform/provider/managed-kind-map.json`, `platform/provider/provider-grafana.yaml`, `scripts/validate.sh`, `docs/` and `platform/function/install.yaml`. A second writer on any of them is a design error, not a merge conflict to resolve. The publishing sequence remains an exclusive resource.
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
