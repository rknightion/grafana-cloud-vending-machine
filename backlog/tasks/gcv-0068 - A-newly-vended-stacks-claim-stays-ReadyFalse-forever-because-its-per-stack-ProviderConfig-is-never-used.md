---
id: GCV-0068
title: >-
  A newly vended stack's claim stays Ready=False forever, because its per-stack
  ProviderConfig is never used
status: Parked
assignee:
  - '@codex'
created_date: '2026-09-10 20:15'
updated_date: '2026-09-12 13:08'
labels:
  - bug
dependencies: []
ordinal: 68000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every newly vended GrafanaCloudStackRequest reports Ready=False with "Unready resources: provider-config" indefinitely, even though the stack and every managed resource under it are Synced=True Ready=True and the credentials are delivered. It clears only when something is provisioned INSIDE that stack, which for many consumers is never.

Found in portina-iac (PTI-0020) on 2026-09-10 against the composition at v1.0.0, comparing two claims in the same namespace on the same composition:

  portina           Ready=True  Available
  portinapushtests  Ready=False Creating   "Unready resources: provider-config"

THE CAUSE. The Grafana provider writes a Ready condition onto a ProviderConfig only once a managed resource USES it - the condition arrives with the first ProviderConfigUsage. Crossplane composed-resource readiness then defaults to requiring Ready=True, so an unused ProviderConfig is indefinitely unready and drags the claim aggregate down with it.

  kubectl -n grafana-vending get providerconfigs.grafana.m.crossplane.io portina -o jsonpath={.status}
    {"conditions":[{"type":"Ready","status":"True","reason":"Available"}],"users":7}
  ... same for portinapushtests
    {}

Empty status, no conditions at all.

WHY IT IS STRUCTURAL RATHER THAN A ONE-OFF. A stack is CREATED through the ORG-level ProviderConfig; the per-stack ProviderConfig the composition also creates is only used by resources provisioned inside that stack. Confirmed by which config each managed resource references in that cluster:

  grafana-cloud-org -> 7 AccessPolicy, 7 AccessPolicyRotatingToken, 2 Stack,
                       2 StackServiceAccount, 2 StackServiceAccountRotatingToken
  portina           -> 3 Dashboard, 3 Folder, 1 OrganizationPreferences
  portinapushtests  -> nothing

portina looks healthy only because it happens to have dashboards, folders and org preferences vended into it. A stack vended purely as a telemetry destination - which is a normal and intended use - has no such resources and so never goes Ready.

A reconcile nudge does not help, and that is diagnostic rather than incidental: the missing thing is a ProviderConfigUsage, not a reconcile. Annotating the claim changed nothing.

WHY IT MATTERS BEYOND COSMETICS. A permanently unready claim is indistinguishable in kubectl output from a genuinely failed vend, so it trains operators to ignore the readiness column - and anything gating on claim readiness (a CI wait, an Argo CD health check, an app-of-apps sync wave) blocks forever on a stack that is completely functional. In portina-iac the claim sat unready for nearly three hours while the stack, both AccessPolicies, both rotating tokens, the service account, its token, three ExternalSecrets, three PushSecrets and three Secrets Manager containers were all healthy.

TWO CANDIDATE FIXES, both composition-side, and the choice is an owner call rather than obvious:
1. Do not require the per-stack ProviderConfig to be in use for readiness - exclude it from the readiness roll-up, or give it an explicit readiness check that does not depend on a usage appearing.
2. Provision one trivial in-stack resource as part of every vend (an OrganizationPreferences or a single folder), so a usage always exists. This makes the readiness honest but adds a resource every consumer then owns.

Option 1 is probably right: the per-stack ProviderConfig genuinely IS ready in the sense that matters - it holds valid credentials and would work the moment anything used it - so requiring proof-of-use is measuring the wrong thing.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A stack vended with no in-stack resources reaches Ready=True
- [ ] #2 The chosen approach is recorded, including why the per-stack ProviderConfig readiness is or is not part of the roll-up
- [ ] #3 An existing vended stack that is currently stuck unready reaches Ready without being recreated
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Lane C adds the focused provider-config readiness tests first.
2. Mark provider-config ready only after observed existence, preserving every other child readiness result.
3. Root runs the integrated gate and records exact-SHA hosted evidence; AC3 is answered from source and marked not exercisable here.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.
<!-- SECTION:FINAL_SUMMARY:END -->
