---
id: GCV-0077
title: >-
  Composed resources churn hard enough to hold the composite watch circuit
  breaker open on every stack
status: To Do
assignee: []
created_date: '2026-09-16 16:48'
labels: []
dependencies: []
references:
  - >-
    https://github.com/crossplane/crossplane/blob/main/internal/circuit/token_bucket.go
  - >-
    https://github.com/crossplane/crossplane/blob/main/internal/circuit/mapfunc.go
type: bug
ordinal: 77000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every composite on both estates running this platform reports Responsive=False with reason WatchCircuitOpen. This is Crossplane's own protection working, not a Crossplane bug: the composite controller wraps its composed-resource watches in a token bucket breaker whose defaults allow a burst of 50 updates and one update every 2 seconds sustained, stays open for 5 minutes, and lets a probe through every 30 seconds. Something is updating composed resources fast enough to exhaust that budget continuously.

Observed live on 2026-09-16. Objects are rewritten every few seconds with byte-identical content: two snapshots 25 seconds apart differed only in resourceVersion, with spec, status, conditions and managedFields all unchanged, and metadata.generation was stable, so this is not the composition re-applying a changed spec.

The affected kinds are specific and the pattern is worth starting from. AccessPolicy churns on both estates. On the estate that also vends in-stack content, FolderPermission and DashboardPermission churn by roughly seven times as much as anything else, followed by Folder and Dashboard. Stack, StackServiceAccount, Team, Role, RoleAssignment, RoleAssignmentItem, OrganizationPreferences and PluginInstallation never trigger the breaker at all. The kinds that churn are the ones carrying unordered list fields (access policy scopes, permission lists) and the ones that do not carry such fields stay quiet, which points at list-ordering instability between what the composition sets and what the API returns, producing a diff on every observe. That is a hypothesis from the correlation, not a proven cause.

Impact today looks bounded: event-driven composite reconciles are throttled to one probe per 30 seconds while the breaker is open, and poll-driven reconciles continue, so nothing is stuck. The costs are slower convergence after a real change, continuous API churn against the vendor, and a Responsive=False condition on every composite, which makes a genuinely degraded composite indistinguishable from the normal state.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The source of the repeated no-op updates is identified per affected kind, with evidence showing what the provider sends and what the API returns
- [ ] #2 Whether the churn originates in the provider or in what this composition sets is settled, and the finding names which of the two has to change
- [ ] #3 Either the churn is removed for the kinds this platform composes, or the platform records why it is acceptable, what it costs and what it would take to fix
- [ ] #4 A composite with no failing children reports Responsive=True, or the condition is documented as uninformative with the reason
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
