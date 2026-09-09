---
id: GCV-0061
title: >-
  Let a second cluster mint credentials against, and adopt, a stack it did not
  vend
status: Done
assignee:
  - campaign-root
created_date: '2026-09-09 11:26'
updated_date: '2026-09-09 22:09'
labels: []
dependencies: []
priority: high
ordinal: 61000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Today the vending machine assumes one Crossplane installation owns a stack end to end: the `GrafanaCloudStackRequest` claim creates the stack, and the `AccessPolicy` / `AccessPolicyRotatingToken` CRs that mint credentials against it resolve `realm.stackRef.name` against a claim object in the SAME cluster. There is no demonstrated way for a second cluster to mint credentials against a stack another cluster vended.

This blocks a real migration shape: a tenant whose workloads move off a shared cluster onto a dedicated cluster in a separate network, while its stack stays where it was vended. The new cluster needs to mint its own `metrics:write` / `logs:write` / `traces:write` and `sourcemaps` credentials against that same stack.

Two distinct capabilities are wanted, and they should be separable:

1. **Credential minting against a stack this cluster did not vend.** An `AccessPolicy` whose realm names a stack by slug or id rather than by local claim reference, so a consumer cluster can mint scoped tokens without owning the stack's lifecycle.
2. **Adoption of an existing stack by a new claim.** So stack ownership can be MOVED between clusters rather than only created. `spec.lifecycle.externalResources: Retain` already means deleting a claim leaves the Grafana stack alive - but a fresh claim for a slug that already exists will attempt a create, and it is unverified what happens: it may conflict, or silently adopt, or create a duplicate. That behaviour needs establishing before any migration relies on it, because getting it wrong risks a live stack.

GCV-0016 added an observe-only stack inventory API for drift detection and adoption, which may already be most of capability 2 - check whether it covers claim-side adoption or only read-side inventory before designing anything new.

Interim position for a migration of this shape: de-provision the stack from the old cluster and re-provision it in the new one, accepting whatever that costs, rather than waiting on this feature.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 An AccessPolicy can name its realm stack by slug or id, without requiring a local GrafanaCloudStackRequest claim for that stack
- [x] #2 A cluster holding no local claim for a stack admits an AccessPolicy naming that stack and renders a rotating-token child bound to it, proven against the real API server; live minting is outside this repository's no-live-contact constraint and is recorded as downstream proof
- [x] #3 The behaviour of a new GrafanaCloudStackRequest for an already-existing slug is established and documented: conflict, adopt, or duplicate
- [x] #4 If adoption is supported, moving a stack between clusters is documented as an ordered procedure, including what happens to the generated-credentials secret path
- [x] #5 GCV-0016's observe-only inventory API is assessed for overlap and either reused or explicitly ruled out with a reason
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 9: freeze provider-observed stack identity with platform-bound target authorization; lanes A schema, B renderer, C inert catalog, D pinned-source adoption and ordered migration; root wiring and real admission tests; lane F security review; two CodeRabbit passes; local gate, publish/signature verification/repin, exact-SHA hosted gate and final reconciliation.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 9 security review approved with historical findings and no unresolved code finding. Two CodeRabbit passes returned five minor findings: four fixed, one redundant omitted-ceiling case declined with accessor-equivalence evidence. One temporary file-ownership violation was restored byte-identically; one unscoped client dry-run has unknown schema-read contact and zero reported server mutations. Review hash-method limitations are preserved in the wave 9 report. Live behavior remains unproven.

Downstream outcome, 2026-09-09: an infrastructure migration in a private source environment motivated this task, and it did NOT exercise adoption. The repository owner's decision there was to delete the existing Grafana Cloud stack from the old cluster and let the new cluster's claim vend a fresh one, which is the interim position this task's description already records. So capability 2 (adoption of an existing stack by a new claim) remains unexercised against a real provider - it is established and documented here, not proven by that migration. Capability 1 (credential minting against a stack this cluster did not vend) was likewise not needed, because the new cluster vends its own stack. Nothing here reopens the task; this note exists so a later reader does not cite that migration as live evidence for either capability.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Delivered independent GrafanaStackConsumer admission, Observe-only stack identity resolution, stack-scoped access policy, observed-policy-ID rotating token and platform-owned credential output without a local stack claim. Platform profiles bind namespace, target, provider, scopes and unique credential/output identities. Real API-server tests install 23/23 CRDs and admit 34 catalog requests; one non-XRD Composition input is excluded, not skipped. AC3 is unchanged: pinned provider/runtime source supports adopt and reconcile, explicitly unproven against live behaviour. docs/migration-1.0.md provides the full source chain, ordered stop-all-source-writers migration and output ownership handoff; in-stack inventory is explicitly ruled out for Cloud stack ID/adoption. Retention, live minting, rotation and migration need downstream proof. Acceptance completing commit 3314fb91bed68f4a4d3fe48b5c469a45a270f85a; hosted Validate public reference run 34388031402 succeeded. Local just check passed: 505 assertions, zero failures, zero skips, 84.9% coverage. Published function source 15b4df59f51391b1f492110276211d0b929c4c9d, verified digest sha256:e5b10d21e462e8892f2865a26cd263e4b576278ad4fa842c485ccd12233db2e9; wrong source SHA rejected before exact-source verification. Final tracker bookkeeping SHA is certified separately in the wave 9 terminal report.
<!-- SECTION:FINAL_SUMMARY:END -->
