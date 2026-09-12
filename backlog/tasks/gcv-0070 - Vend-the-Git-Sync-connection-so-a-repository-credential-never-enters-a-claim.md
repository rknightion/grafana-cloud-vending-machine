---
id: GCV-0070
title: Vend the Git Sync connection so a repository credential never enters a claim
status: Parked
assignee:
  - '@codex'
created_date: '2026-09-12 12:44'
updated_date: '2026-09-12 15:59'
labels: []
dependencies: []
type: feature
ordinal: 70000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GrafanaProvisioningRepository references a Grafana connection by name and deliberately never manages it, so every consumer has to create that connection out of band before a Git-provisioned subtree can authenticate. The connection managed kind is activated on the provider but has no composite API, no composition and no renderer, which makes the one prerequisite of the Git Sync module the only part of it that is not vendable.

The provider kind is present at the current pin and its spec carries title, type, description, url and a GitHub App block of app id plus installation id, alongside a write-only secure block and a secure version counter.

The design question the module exists to answer is where the credential lives. The secure block takes a map per key with two accepted shapes: a create key carrying the credential inline, and a name key referencing a secure value that already exists in the stack. Only the second shape is expressible from a git-tracked claim in this repository, because the first puts a bearer-equivalent literal into a manifest and into cluster storage. Multiple connections per stack are a normal arrangement, since a stack can sync subtrees from more than one source.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A namespaced composite API vends a provider connection carrying title, type, description, url and the GitHub App app id and installation id
- [ ] #2 The credential is expressible only as a reference to a secure value that already exists in the stack, and a claim carrying an inline credential literal is rejected at admission rather than at reconcile
- [ ] #3 The secure version counter is a claim input, so a rotation is triggerable without recreating the connection
- [ ] #4 Several connections in one namespace are provable from one claim set, each with its own stable external name
- [ ] #5 The pinned provider connection CRD is added to the provider admission fixture from the pinned artefact, and the emitted shape round-trips through it
- [ ] #6 GrafanaProvisioningRepository can reference a connection this API vends, and the catalog documents which of the two is applied first
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Root imports the pinned Connection and SecureValue CRDs after a byte-equality provenance control.
2. Lane A implements the frozen connection API, three-resource credential bridge, fail-closed admission controls, renderer tests, and catalog.
3. Root wires registries, runs the integrated gate and security review, then records exact-SHA hosted evidence.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.

Additional resume check: the cached securevaluev1beta1s.enterprise.grafana.m.crossplane.io CRD declares spec.names.kind and listKind as SecurevalueV1Beta1 and SecurevalueV1Beta1List, while the frozen wave seam spells the kind SecureValueV1Beta1. Do not emit or register either spelling until the package provenance blocker is resolved and the actual admitted GVK is re-frozen.

Reviewer correction, 2026-09-12, after wave 10 parked. The provenance blocker is cleared and the fixture pre-pass has landed on main at 8b183cd; hosted Validate public reference run 34702990499 succeeded at that SHA. ConnectionV0Alpha1, RepositoryV0Alpha1, SecurevalueV1Beta1 and SlackChannel are now in the admission fixture, each proven identical to its copy in the package at the pinned digest.

Wave 10 was right that the control failed and wrong about what that meant. The control demanded byte equality against an existing fixture neighbour, and the fixture is deliberately inconsistent about an empty top-level status stanza: 12 of its 42 entries carry one and 30 do not, and the installer ignores it. The real provenance evidence is that all 42 pre-existing fixture CRDs are present in the cached package and every one agrees exactly modulo that stanza.

Two frozen names were wrong, not one. Wave 10 caught the kind spelling and missed the group version. Both are now read off the artefact: kind SecurevalueV1Beta1 with a lowercase v, listKind SecurevalueV1Beta1List, plural securevaluev1beta1s, singular securevaluev1beta1, scope Namespaced, and served group version enterprise.grafana.m.crossplane.io/v1alpha1. The V1Beta1 in the kind and plural is the Grafana app-platform resource version, not the CRD version, so a managed-kind-map entry or a rendered child written against /v1beta1 is rejected.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.

Additional resume check: reconcile the cached CRD kind SecurevalueV1Beta1 with the frozen SecureValueV1Beta1 seam before implementation.
<!-- SECTION:FINAL_SUMMARY:END -->
