---
id: GCV-0044
title: Vend the alerting notification routing tree
status: Done
assignee: []
created_date: '2026-09-08 22:35'
updated_date: '2026-09-09 00:53'
labels: []
dependencies: []
priority: high
type: feature
ordinal: 44000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GrafanaAlertingBundle vends contact points, message templates, mute timings and rule groups, but the notification policy tree is not vended at all. Its existing rule groups use per-rule notificationSettings that deliver directly to their contact points and bypass the notification policy tree. This task adds platform-owned policy routing for ordinary rules; the previous manual-wiring claim was incorrect. The pinned provider carries the resource: alerting notificationpolicies and routingtreev1beta1 are both present at v2.14.0 and neither appears in the emitted-kind activation map.

The routing tree is a whole-set resource: writing it replaces the entire policy tree for the org. This repository has already shipped one defect of exactly that class, so the ownership question is the design question here, not an afterthought - see the whole-set-resources rule in the Wave operating model document.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The notification routing tree is vended per stack from platform-controlled input, and a vended contact point is reachable through it without manual wiring
- [x] #2 The XRD refuses a request that would leave a vended contact point unroutable, with its own rejection message, proven against the real API server
- [x] #3 Exactly one declarative owner writes the tree; the design records what happens to policy tree content the platform did not vend, and that behaviour is tested
- [x] #4 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [x] #5 A catalog example renders inert and is covered by the catalog README and Kustomization checks
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root source correction: alerting.go always supplies per-rule notificationSettings. Existing bundled rules already select their contact point directly; lack of NotificationPolicy does not make that contact point unreachable. New policy-routed rules have a separate remote identity and omit the bypass.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Vended one platform-owned whole notification policy tree per stack with ContactPoints and ordinary policy-routed RuleGroups. Real API-server create/update and weaken/admit/restore prove both contact reachability and route references. The legacy bundle already has per-rule direct routing and retains that behavior. Completing source/pin SHA: 187b03ea40ee32fcea890e40c138f00a8c73bd5f. Hosted Validate run 34296536930: success. Local just check passed with 23 real API-server tests, zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
