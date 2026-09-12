---
id: GCV-0070
title: Vend the Git Sync connection so a repository credential never enters a claim
status: To Do
assignee: []
created_date: '2026-09-12 12:44'
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
