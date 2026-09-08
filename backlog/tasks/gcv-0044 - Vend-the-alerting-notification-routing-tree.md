---
id: GCV-0044
title: Vend the alerting notification routing tree
status: To Do
assignee: []
created_date: '2026-09-08 22:35'
labels: []
dependencies: []
priority: high
type: feature
ordinal: 44000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GrafanaAlertingBundle vends contact points, message templates, mute timings and rule groups, but the notification policy tree is not vended at all. Alerts therefore evaluate and fire into a routing tree the platform does not control, so a vended contact point is only reachable if somebody wires it by hand in the UI. The pinned provider carries the resource: alerting notificationpolicies and routingtreev1beta1 are both present at v2.14.0 and neither appears in the emitted-kind activation map.

The routing tree is a whole-set resource: writing it replaces the entire policy tree for the org. This repository has already shipped one defect of exactly that class, so the ownership question is the design question here, not an afterthought - see the whole-set-resources rule in the Wave operating model document.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The notification routing tree is vended per stack from platform-controlled input, and a vended contact point is reachable through it without manual wiring
- [ ] #2 The XRD refuses a request that would leave a vended contact point unroutable, with its own rejection message, proven against the real API server
- [ ] #3 Exactly one declarative owner writes the tree; the design records what happens to policy tree content the platform did not vend, and that behaviour is tested
- [ ] #4 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [ ] #5 A catalog example renders inert and is covered by the catalog README and Kustomization checks
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
