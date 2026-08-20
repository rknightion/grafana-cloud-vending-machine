---
id: GCV-0006
title: Give the API a reviewable decommission path
status: Done
assignee:
  - '@codex'
created_date: '2026-08-20 16:10'
updated_date: '2026-08-20 17:50'
labels:
  - api
  - design
dependencies: []
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
managementPolicies (Create, Observe, Update, LateInitialize) and the Stack's deleteProtection: true are package-level constants in the function with no API surface. Deleting a stack for real therefore cannot be expressed as a request at all.

What it takes today, proven on a live control plane retiring two stacks: annotate each request crossplane.io/paused=true so the composition cannot revert the next step; patch every org-level managed resource to managementPolicies ['*']; patch the Stack's forProvider.deleteProtection to false; wait for status.atProvider to confirm protection is off externally rather than only in spec; remove the request from Git; then remove the pause annotation so the finalizer clears.

Every one of those steps is a kubectl call against a composed resource. None appears in a diff, none is reviewable, and the operator has to work out for themselves which resources hold external state that outlives the stack. Get that last part wrong and the visible outcome is a clean deletion while an organization-level access policy and its live telemetry-publisher token stay behind with nothing owning them. Stack-local resources are safe to orphan because deleting the stack takes them; organization-level ones are not, and nothing in the repository says so.

The principle that there is no one-command destructive path is worth keeping. A lifecycle field does keep it: arming deletion is one reviewable change and removing the request is a second, and both are visible to a reviewer, which the current sequence is not.

Design questions to settle first: whether the field belongs on the request or is inferred from the profile; whether deleteProtection and management policies move together or separately; whether an armed-but-not-deleted request should surface a condition; and what happens to generated secret-store documents, which today are retained by deletionPolicy and were deleted by hand.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A platform-owned lifecycle field on GrafanaCloudStackRequest selects between retaining and deleting external resources, defaulting to retain
- [x] #2 Which profiles may select deletion is a platform decision, not a consumer one
- [x] #3 Arming deletion and removing the request stay separate changes, so there is still no one-command destructive path
- [x] #4 The decommission runbook is rewritten around the field, and the manual patching sequence is deleted
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Map every external-resource lifecycle and settle the platform-owned arming contract.
2. Add test-first function behavior and the request/profile API surface with retain as the default.
3. Rewrite decommission documentation, validate, review, publish, pin, and record exact-SHA hosted proof.

4. Security hardening: replace the reserved ESO tag, bind Delete authorization to the exact request identity and immutable profile, and require current ESO finalizer/sync evidence before deletionReady.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Design frozen 2026-08-20: add spec.lifecycle.externalResources with Retain/Delete and Retain default. Delete is rejected unless the selected spec.profile appears in platform-owned Composition input deletionAllowedProfiles, which defaults empty. Armed mode changes only state that can outlive the stack: Stack delete protection and Delete management, stack administrator service account/token, telemetry access policy/token, and administrator/telemetry PushSecret documents. Stack-local resources remain retain/orphan because deleting the stack removes their external state. status.deletionReady becomes true only after the observed Stack reports deleteProtection=false. Arming and request removal remain separate reviewed Git changes; the removal change also removes dependent access claims. AWS PushSecret deletion requires DeleteSecret permission and defaults to a 30-day recovery window.

Security review correction 2026-08-20: deletionAllowedProfiles was rejected as an authorization boundary because spec.profile is consumer-selected. The implementation now uses platform-owned deletionAuthorizations entries bound to namespace, request name, and immutable profile; the reference list remains empty. deletionReady additionally requires observed deleteProtection=false plus deletionPolicy=Delete, the ESO deletion finalizer, a current-generation syncedResourceVersion, non-empty syncedPushSecrets, and Ready=True for the administrator and any enabled telemetry PushSecret. The reserved ESO managed-by tag was replaced with the project-owned grafana-cloud-vending-machine=managed tag, with matching AWS IAM condition.

Second security correction 2026-08-20: deletionAuthorizations now includes the request UID, preventing replay against a later request incarnation. Existing rotating tokens fall back to their observed parent IDs during transient parent status loss; deletionReady now also requires both enabled token resources to observe managementPolicies ['*'] and deleteOnDestroy true. New access claims fail closed when the referenced stack reports deletionArmed or has a deletionTimestamp, while one-way admission retains already-observed children for deliberate Stage 2 removal.

CodeRabbit correction audit 2026-08-20: dismissed two Major suggestions after source verification. ESO v2.6.0 status.syncedResourceVersion is generation-hash, not metadata.resourceVersion, so the current-generation prefix check is correct. Gating destructive desired fields on deletionReady would be circular because readiness must first observe those fields; Stage 1 intentionally renders deletion-capable policies without deleting resources, and readiness proves their observed preparation before removal.

Local implementation evidence 2026-08-20: focused lifecycle/security RED-GREEN tests and full go test -race ./... passed; go vet ./... passed; fresh SECURITY review returned PASS with no Critical or Warning findings; CodeRabbit's two code suggestions were dismissed against pinned ESO/controller semantics, and its final two tracker-assignee findings were false positives for the repository's generic @codex actor. ./scripts/validate.sh passed with 89.3% statement coverage and the public-release scan passed.

Published package evidence 2026-08-20: source/merge SHA d2343aef13dac0a41505d35da6ca2545ae1ed7df passed hosted Validate public reference run 32399213024. Publish Grafana vending function run 32399213015 succeeded; immutable amd64/arm64 package digest sha256:fb5e86a7a664572ef3383da16e85f1468c6d13ac8fd9abff61268daeb5bc44b8 verified with Cosign against https://github.com/rknightion/grafana-cloud-vending-machine/.github/workflows/publish-function.yml@refs/heads/main and is being pinned in the delivery commit.

Completion evidence: SHA d8d8ee9d90d714249178e468f908721e805a6af8; hosted Validate public reference run 32399726607 passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added a retain-default, exactly authorized and observable decommission lifecycle; hardened rotating-token and ESO deletion preparation; rewrote the three-stage runbook; published, signed, and pinned the multi-platform function package. Local validation and exact-SHA hosted validation passed.
<!-- SECTION:FINAL_SUMMARY:END -->
