---
id: GCV-0018
title: Add a per-stack alerting bundle with an explicit provenance decision
status: To Do
assignee: []
created_date: '2026-08-21 12:15'
labels: []
dependencies: []
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a single-owner per-stack alerting composite covering RuleGroup, ContactPoint, MuteTiming, MessageTemplate and InhibitionRuleV1Beta1. Contact points are already an optional stack child, so this generalises that rather than replacing it.

Exclude NotificationPolicy. It is an organization-wide singleton whose provider documentation states it manages the entire tree and overwrites its policies, and whose deletion resets the tree to defaults. In a stack-per-tenant topology one Crossplane owner per stack is coherent, so the objection is not ownership but blast radius on delete and the absence of any partial form. Route instead through per-rule simplified routing, which is available now, and revisit named routing trees later: RoutingtreeV1Beta1 is preview, requires Grafana 13.1 or later and a feature toggle, and no resource was found that sets Grafana feature toggles on a Cloud stack.

The decision this API must not default is provenance. Resources provisioned through the alerting provisioning API are stamped with a provenance marker and become read-only in the tenant Grafana UI. Setting the disable provenance flag hands editing back and accepts that Crossplane will contest tenant edits on every reconcile. There is no third option, so make it a required enum on the spec rather than inheriting whatever the provider defaults to. Handoff is a legitimate posture for seeded content while security stays locked.

Note the existing reconciliation vocabulary already distinguishes enforced from createOnly; align naming with it rather than inventing a parallel scheme.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The spec carries a required provenance choice and neither value is silently defaulted
- [ ] #2 NotificationPolicy is not rendered by any path, and the README states why and what to use instead
- [ ] #3 Naming aligns with the existing enforced and createOnly reconciliation vocabulary
- [ ] #4 Exactly one owner per stack for each whole-set alerting object
- [ ] #5 A catalog example renders a rule group with simplified routing and no live endpoints
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
