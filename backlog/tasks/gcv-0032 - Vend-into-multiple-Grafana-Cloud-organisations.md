---
id: GCV-0032
title: Vend into multiple Grafana Cloud organisations
status: To Do
assignee: []
created_date: '2026-09-08 08:08'
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
