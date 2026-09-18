---
id: GCV-0077
title: >-
  Composed resources churn hard enough to hold the composite watch circuit
  breaker open on every stack
status: Parked
assignee: []
created_date: '2026-09-16 16:48'
updated_date: '2026-09-18 10:22'
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
- [x] #1 The source of the repeated no-op updates is identified per affected kind, with evidence showing what the provider sends and what the API returns
- [x] #2 Whether the churn originates in the provider or in what this composition sets is settled, and the finding names which of the two has to change
- [x] #3 Either the churn is removed for the kinds this platform composes, or the platform records why it is acceptable, what it costs and what it would take to fix
- [ ] #4 A composite with no failing children reports Responsive=True, or the condition is documented as uninformative with the reason
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 13: derive candidate churn fields from the renderer and pinned CRD without live contact, test-first the permission-kind correction if evidenced, and return any AccessPolicy edit as a wiring packet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-09-18 live re-verification against both estates. Two claims in the description are WRONG and one is confirmed. Read this before starting.

CONFIRMED: every composite on both estates still reports Responsive=False with reason WatchCircuitOpen, and the affected kinds are exactly the three named. Evidence is metadata.generation, which increments only on a SPEC write, so it counts spec rewrites directly and is a far better instrument than sampling resourceVersion.

Estate with in-stack content, one object of each kind:
  DashboardPermission  generation 518499
  FolderPermission     generation 517384
  AccessPolicy         generation 202187
  Folder 2, Dashboard 3, Team 2 and 3, OrganizationPreferences 2, Stack 3

Estate without in-stack content, seven access policies and two stacks:
  AccessPolicy         generation 191, 191, 191, 79963, 81197, 82591, 85959
  Stack                generation 3

So the quiet kinds sit at 2 or 3 generations and the churning kinds sit between 80 thousand and half a million. The ratio is roughly 170000 to 1, and the permission kinds do churn about 2.6 times as much as AccessPolicy, which the description's 'seven times' overstates but gets the direction right.

WRONG 1: 'metadata.generation was stable, so this is not the composition re-applying a changed spec.' Generation is not stable, it is enormous. The 25-second window it was sampled in was too short, and at least one object is quiet for 20 seconds at a time, so the churn is bursty per object rather than continuous. Something IS rewriting the spec, tens of thousands of times. A composed resource's spec is written by the composite controller from what the composition function renders, so the description's own AC2 question now leans hard toward this repository rather than the provider - but which field is still open.

WRONG 2: the list-ordering hypothesis is FALSIFIED for AccessPolicy. Scope ordering does not correlate with churn, and it runs the wrong way:
  generation 85959 - spec scopes NOT sorted, atProvider sorted
  generation 82591 - spec scopes ALREADY sorted, atProvider identical
  generation 81197 - spec scopes NOT sorted
  generation 79963 - spec scopes ALREADY sorted, atProvider identical
  generation 191   - spec scopes NOT sorted
Two of the four highest-churn policies already emit their scopes in exactly the order the API returns, and the lowest-churn object emits them unsorted. Sorting the scopes list will therefore not fix this. Do not start there, and do not re-derive the hypothesis: it has been tested and it is dead.

THE DISCRIMINATOR TO START FROM instead. On the estate with no in-stack content, four AccessPolicy objects churn and three do not, same kind, same cluster, same provider. The three quiet ones sit at exactly 191 generations each, which is suspiciously equal and suggests they stopped rather than never started. Diff a churning object against a quiet one field by field - spec, status.atProvider, managedFields ownership - and the differing field is the cause. That comparison is cheap and it is the whole job.

One more structural clue worth testing: all three churning kinds are WHOLE-SET replace resources, the ones the wave operating model already singles out as needing exactly one declarative owner. The quiet kinds are not. Two writers fighting over a whole-set field would produce exactly this signature.

Also observed, and separately relevant to GCV-0075 AC4: AccessPolicyRotatingToken exposes NO expiration, secondsToLive or earlyRotationWindowSeconds in status.atProvider at all - all three read null - while StackServiceAccountRotatingToken exposes every one of them. An operator cannot see when an access policy token expires from the resource.

Wave 13 terminal reconciliation: pinned-source evidence identifies resolver and late-initializer scalar writes as the no-op spec-write source for AccessPolicy, FolderPermission, and DashboardPermission. Focused tests prove the renderer preserves observed identifiers, UIDs, team IDs, and org IDs. No live cluster contact was permitted, so generation settling and Responsive=True remain unverified.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Parked after landing the source-backed renderer correction. The composition now preserves provider-resolved fields for all three affected kinds, with focused tests and integrated just check plus hosted Validate run 35333659147 green at b47831ddddfd5ec10e1e699d4d8608886261becd. Resume after deploying this pin: observe generations over several burst windows and confirm a healthy composite returns Responsive=True. AC4 remains unproven.
<!-- SECTION:FINAL_SUMMARY:END -->
