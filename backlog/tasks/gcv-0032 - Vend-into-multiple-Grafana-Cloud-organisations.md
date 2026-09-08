---
id: GCV-0032
title: Vend into multiple Grafana Cloud organisations
status: In Progress
assignee: []
created_date: '2026-09-08 08:08'
updated_date: '2026-09-08 11:10'
labels: []
dependencies: []
priority: high
type: feature
ordinal: 32000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every stack request today lands in exactly one Grafana Cloud organisation. organizationProviderConfigName is a single scalar in the stack Composition pipeline input, read once into platformSettings and used for every organisation-plane child: the Stack itself, its service account, the rotating token, the stack-realm access policies and plugin installations. The XRD also carries enforcedCompositionRef, so a second Composition is not available as a workaround. One organisation credential, one remote secret path, no selector anywhere.

A platform team running more than one Grafana Cloud organisation therefore cannot use this reference at all, and organisation choice is not a property a stack can acquire later: a Cloud stack belongs to the organisation that created it and cannot be moved.

Note the constraint recorded on the promotion-ladder task is about a stack belonging to one organisation, which stays true. It does not say a vending machine cannot serve several, and the README must not read as though it does.

The organisation is a platform-owned routing decision, not free-form request input, so the request names an organisation and the platform resolves it against a registry it owns. An unknown name must fail closed rather than fall through to a default.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 spec.organization is a required, immutable field on the stack request, and the XRD rejects a change to it
- [ ] #2 The Composition input carries an organisation registry keyed by organisation name, each entry naming a ProviderConfig, an allowed-regions list and an allowed-usages list
- [ ] #3 The function fails closed on an unknown organisation name, on a region not allowed for the named organisation, and on a usage not allowed for it, rather than defaulting
- [ ] #4 Every organisation-plane child resolves its ProviderConfig through the registry entry, with no remaining path to a single hard-coded organisation ProviderConfig
- [ ] #5 The remote secret path carries an organisation segment, and the example AWS IAM policy scopes Secrets Manager access by that prefix
- [ ] #6 Every catalog example carries the new required field and renders
- [ ] #7 The README documents the organisation registry as platform-owned and states that a stack cannot change organisation after creation
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 2 root plan: re-pin and verify the provider release identity; implement the required immutable organization seam, platform registry, fail-closed routing, organization-segmented secret path, catalogs, AWS examples, shared product/profile/input seams, split API registries, and pre-register compiling renderer stubs; run the full gate; commit and push with explicit pathspecs before dispatching lanes A-J.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 1 disposition, 2026-09-08: Parked before implementation. The requested operational root route was gpt-5.6-sol at high effort, but the generic spawn exposed only its task name and no role, model, or effort metadata. Section 4.1 requires a hard stop when route metadata is missing. The actual clean starting head was 9af868e2b574bb13da11fbf81d81407593cb8366, aligned with origin/main, and hosted validation run 34213680492 passed at that exact SHA. Resume only in a client session that exposes and confirms the root model and effort; then re-read the wave goal, reconcile the recorded starting-state drift, and begin at section 5.0 step 2. No acceptance criterion or Definition of Done item was checked, and no implementation file or external service was changed.
<!-- SECTION:NOTES:END -->
