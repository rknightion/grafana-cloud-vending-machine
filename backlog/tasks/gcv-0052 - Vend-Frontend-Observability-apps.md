---
id: GCV-0052
title: Vend Frontend Observability apps
status: In Progress
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-08 22:48'
labels: []
dependencies: []
type: feature
ordinal: 52000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
frontendobservability apps is a three-CRD group in the pinned provider with nothing vended. It is the Faro surface: a vended stack cannot receive browser telemetry until somebody creates an app by hand and pastes its key into a frontend build.

Small and self-contained, which is why it is tracked separately rather than folded into a larger surface. The only real design question is credential handling: the app key is what a browser bundle carries, so the machine has to be explicit about what is published and what is not.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Frontend observability apps are vended per stack from platform-controlled input, with an explicit external name so a re-adopt does not attempt a create
- [ ] #2 The design records exactly which app credential is published and why a browser-visible key is not treated as a secret, and a test pins that boundary
- [ ] #3 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [ ] #4 A catalog example renders inert and is covered by the catalog README and Kustomization checks
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.
<!-- SECTION:PLAN:END -->
