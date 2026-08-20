---
id: GCV-0003
title: Stage stack-local resources so a healthy first vend is quiet
status: To Do
assignee: []
created_date: '2026-08-20 16:09'
labels:
  - function
  - reconciliation
dependencies: []
type: bug
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every stack-local resource reaches the stack through the per-stack ProviderConfig, whose credential Secret does not exist until the admin token has been minted and published to the secret store; plugin installation calls the stack's own API through the organization credential. All of them were rendered on the first pass, against a stack that was not serving yet.

The vend still converged, because Crossplane retries, but it emitted an error per resource per reconcile. Measured on one clean, entirely successful vend of a comprehensive-catalog request: 59 warning events across 9 reasons, led by CannotConnectToProvider x14 ('cannot get terraform setup: cannot extract credentials: cannot get credentials secret'), CannotResolveResourceReferences x8, CannotCreateExternalResource x8 and CannotUpdateManagedResource x6. It was also enough churn to open the composite's own watch circuit breaker: Responsive=False WatchCircuitOpen, 'Too many watch events from PluginInstallation'.

Three costs, in order of importance: a genuine failure is indistinguishable from ordinary creation noise; anything alerting on managed-resource conditions has to whitelist the first several minutes of every vend; and a first-time operator watching a successful provision sees a wall of red.

Gate each group on the condition it actually depends on rather than rendering everything at once. Conditional rendering on observed state is already the idiom used for the rotating token, which waits for the service account's observed id.

Admission must be one-way. A desired resource that disappears is deleted, so a member that already exists has to be kept regardless of the gate, or a transient not-Ready would take a stack's content with it and recreate it -- destructive under enforced dashboard reconciliation.

No status handling is needed: the gate flips and the stage-two resources enter the desired set in the same reconcile, so there is no pass in which the desired set is foundation-only and fully ready. An earlier read of this said the composite would report Ready prematurely and need its status held down. That was WRONG.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Plugin installations render only once the Stack reports Ready
- [ ] #2 Baseline content and the optional domains render only once the per-stack credential ExternalSecret is also Ready
- [ ] #3 A staged resource that already exists is never withdrawn from the desired set
- [ ] #4 Tests assert the resource set either side of the gate
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
