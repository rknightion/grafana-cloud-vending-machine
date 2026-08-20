---
id: GCV-0008
title: Stage the access claims on the stack they reference
status: To Do
assignee: []
created_date: '2026-08-20 16:10'
labels:
  - function
  - reconciliation
dependencies: []
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Depends on GCV-0003, which stages the stack request's own children but cannot reach the access claims. GrafanaCustomRoleBinding, GrafanaTeamAccess and GrafanaContentAccessPolicy are separate composites with their own observed sets, so they cannot see whether the stack named in spec.stackRef is serving or whether its credential Secret exists. On the same measured vend they contributed their own share of the warnings, on names like <slug>-dashboard-readers-role and <slug>-observability-engineers-role.

The mechanism to fix it properly is the function response's extra-resource requirements: a function may ask Crossplane to fetch named resources it does not own and receives them on the next iteration, which would let an access claim gate on the referenced request or its ProviderConfig credential. That is a real design step -- it changes what the function asks for on every reconcile -- so it is deliberately not bundled with the stack-request staging.

Cheaper alternative if extra resources prove awkward: gate on the claim's own observed children having ever reconciled, which is weaker but needs no new inputs.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Access claim children are withheld until the stack they reference can accept them
- [ ] #2 Admission is one-way, matching the stack request
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
