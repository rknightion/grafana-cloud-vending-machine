---
id: GCV-0003
title: Stage stack-local resources so a healthy first vend is quiet
status: Done
assignee: []
created_date: '2026-08-20 16:09'
updated_date: '2026-08-20 16:20'
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
- [x] #1 Plugin installations render only once the Stack reports Ready
- [x] #2 Baseline content and the optional domains render only once the per-stack credential ExternalSecret is also Ready
- [x] #3 A staged resource that already exists is never withdrawn from the desired set
- [x] #4 Tests assert the resource set either side of the gate
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented as two staged groups merged into the desired set by admitStaged: whenStackServes (plugin installations) gated on the observed Stack reporting Ready, and whenCredentialsPublished (baseline folders, dashboards, organization preferences, and the SSO, monthly report and incident domains) gated on that plus the per-stack credential ExternalSecret being Ready. Both gates are one-way -- a member already present in the observed set is admitted regardless -- so a transient not-Ready cannot withdraw and therefore delete a stack's content.

markObservedResourcesReady now shares the observedReady helper instead of inlining its own condition check.

TestMinimalStackRendersBaseline was deleted rather than adapted: it asserted the full resource set with no observed state, which is exactly the behaviour being removed, and the two new tests assert the same set either side of the gate. Nine existing test call sites that assert on stack-local resources now pass foundationReadyObserved(). Coverage went from 89.0% to 89.3%.

No composite status handling was needed. The gate flips and the stage-two resources enter the desired set in the same reconcile, so there is no pass where the desired set is foundation-only and fully ready, and the composite cannot report Ready early. The initial read that this would need the status held down was WRONG.

Published from e2a660f as v0.0.0-e2a660f8a6be, digest sha256:ebb7ab2203d2846a5a954dabfc11a0d3fc70e2279b8017cd2f8e07b25b53a014, pinned in 9a2b73d. Hosted validation run 32390912362 green on that SHA.

Confirmed live on the consumer control plane: the new revision installed Healthy=True with both runtime pods up, the in-cluster PreSync verification passed against the new digest, and the existing request kept all 24 managed resources True/True with no withdrawal, which is the regression that mattered.

NOT yet proven live: the reduction in warning events itself. That needs a fresh vend, and the only stack on this control plane already exists. The unit tests pin the render contract; the event-count claim stays a prediction until a disposable stack is vended -- which, until GCV-0006 lands, also means running the manual teardown again to remove it.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Stack-local resources now wait for the stack they address, so the first pass renders only what the organization credential can create. One-way admission means an existing resource is never withdrawn. Live: new function revision healthy, existing request unaffected, all 24 resources retained. The predicted drop in warning events is untested pending a disposable vend.
<!-- SECTION:FINAL_SUMMARY:END -->
