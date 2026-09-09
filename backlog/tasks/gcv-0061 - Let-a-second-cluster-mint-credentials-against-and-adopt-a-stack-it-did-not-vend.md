---
id: GCV-0061
title: >-
  Let a second cluster mint credentials against, and adopt, a stack it did not
  vend
status: To Do
assignee: []
created_date: '2026-09-09 11:26'
labels: []
dependencies: []
priority: high
ordinal: 61000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Today the vending machine assumes one Crossplane installation owns a stack end to end: the `GrafanaCloudStackRequest` claim creates the stack, and the `AccessPolicy` / `AccessPolicyRotatingToken` CRs that mint credentials against it resolve `realm.stackRef.name` against a claim object in the SAME cluster. There is no demonstrated way for a second cluster to mint credentials against a stack another cluster vended.

This blocks a real migration. Portina is moving from the shared robk EKS cluster to a dedicated cluster in a new VPC (`m7kni/portina-iac`). Its stack `portina` was vended by robk's Crossplane. The new cluster needs to mint its own `metrics:write` / `logs:write` / `traces:write` and `sourcemaps` credentials against that same stack.

Two distinct capabilities are wanted, and they should be separable:

1. **Credential minting against a stack this cluster did not vend.** An `AccessPolicy` whose realm names a stack by slug or id rather than by local claim reference, so a consumer cluster can mint scoped tokens without owning the stack's lifecycle.
2. **Adoption of an existing stack by a new claim.** So stack ownership can be MOVED between clusters rather than only created. `spec.lifecycle.externalResources: Retain` already means deleting a claim leaves the Grafana stack alive - but a fresh claim for a slug that already exists will attempt a create, and it is unverified what happens: it may conflict, or silently adopt, or create a duplicate. That behaviour needs establishing before any migration relies on it, because getting it wrong risks the live stack.

GCV-0016 added an observe-only stack inventory API for drift detection and adoption, which may already be most of capability 2 - check whether it covers claim-side adoption or only read-side inventory before designing anything new.

Interim position for the Portina migration: the stack will be de-provisioned from the old cluster and re-provisioned in the new one, accepting whatever that costs, rather than waiting on this feature.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 An AccessPolicy can name its realm stack by slug or id, without requiring a local GrafanaCloudStackRequest claim for that stack
- [ ] #2 A consumer cluster successfully mints a scoped rotating token against a stack vended by a different cluster, proven live
- [ ] #3 The behaviour of a new GrafanaCloudStackRequest for an already-existing slug is established and documented: conflict, adopt, or duplicate
- [ ] #4 If adoption is supported, moving a stack between clusters is documented as an ordered procedure, including what happens to the generated-credentials secret path
- [ ] #5 GCV-0016's observe-only inventory API is assessed for overlap and either reused or explicitly ruled out with a reason
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
