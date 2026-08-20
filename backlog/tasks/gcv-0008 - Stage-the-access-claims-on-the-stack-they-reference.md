---
id: GCV-0008
title: Stage the access claims on the stack they reference
status: Done
assignee:
  - '@codex'
created_date: '2026-08-20 16:10'
updated_date: '2026-08-20 17:50'
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
- [x] #1 Access claim children are withheld until the stack they reference can accept them
- [x] #2 Admission is one-way, matching the stack request
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Map the Crossplane extra-resource iteration contract and all access renderer gates.
2. Freeze one-way admission semantics and implement test-first with sole ownership of shared tests.
3. Review, validate, publish, pin, and record exact-SHA hosted proof.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Design frozen 2026-08-20 against Crossplane 2.3 and function-sdk-go v0.7.1: every access XR always requests its namespaced referenced GrafanaCloudStackRequest under stable key referenced-stack using rsp.Requirements.Resources and a named selector derived from the access XR API group. request.GetRequiredResource reads the next iteration. Exactly one identity-matching referenced XR with standard Ready=True admits configured children; unresolved, missing, Ready False/Unknown/absent fail closed. Multiple results, identity mismatch, conversion error, or advertised capabilities without CAPABILITY_REQUIRED_RESOURCES are fatal. Admission is one-way through existing admitStaged, retaining only configured candidates already observed. While closed, set desired access XR ReadyFalse to prevent vacuous readiness; once open, normal child readiness resumes. No Secret is fetched, no RBAC expansion is needed, and roles.go/access.go/API YAML remain unchanged. Live iteration and warning reduction remain unproven.

Security hardening 2026-08-20: required-stack admission now closes when the referenced request reports status.deletionArmed=true or metadata.deletionTimestamp is present. Existing observed configured children remain admitted through the established one-way rule so Stage 2 can remove their owners deliberately.

Local implementation evidence 2026-08-20: required-resource selector stability, fail-closed readiness, API-version preservation, one-way admission, decommission admission closure, go test -race ./..., go vet ./..., fresh SECURITY PASS, and integrated ./scripts/validate.sh all passed. Live Crossplane iteration remains unperformed; hosted exact-SHA evidence remains pending.

Published package evidence 2026-08-20: source/merge SHA d2343aef13dac0a41505d35da6ca2545ae1ed7df passed hosted Validate public reference run 32399213024. Publish Grafana vending function run 32399213015 succeeded; immutable amd64/arm64 package digest sha256:fb5e86a7a664572ef3383da16e85f1468c6d13ac8fd9abff61268daeb5bc44b8 verified with Cosign against https://github.com/rknightion/grafana-cloud-vending-machine/.github/workflows/publish-function.yml@refs/heads/main and is being pinned in the delivery commit.

Completion evidence: SHA d8d8ee9d90d714249178e468f908721e805a6af8; hosted Validate public reference run 32399726607 passed. Live Crossplane iteration remains outside this public inert reference implementation.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Staged all access claims on an exact required referenced stack with fail-closed, one-way admission, closed new admission during decommission, and verified the package through local and exact-SHA hosted gates.
<!-- SECTION:FINAL_SUMMARY:END -->
