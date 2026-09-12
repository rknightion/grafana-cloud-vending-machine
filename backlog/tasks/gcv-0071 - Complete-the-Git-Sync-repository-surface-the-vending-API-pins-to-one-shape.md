---
id: GCV-0071
title: Complete the Git Sync repository surface the vending API pins to one shape
status: In Progress
assignee:
  - '@codex'
created_date: '2026-09-12 12:44'
updated_date: '2026-09-12 17:32'
labels: []
dependencies:
  - GCV-0070
type: feature
ordinal: 71000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The provisioning renderer writes a repository whose provider type is the literal github, whose sync block is the literal enabled true, target folder, interval sixty, and which carries no change workflows at all. Everything else the provider kind offers is unreachable from a claim, so a consumer wanting anything other than a read-only GitHub folder subtree on a one-minute poll has to leave the vending machine and drive the stack API directly.

What the pinned provider kind actually offers, and the API does not: a provider type of local, github, github enterprise, git, bitbucket or gitlab, each with its own url, branch and path block plus a token user for the basic-auth forms and a dashboard-preview toggle for the GitHub forms; a sync target of instance, folder or folderless with a settable interval; an allowed-workflow list of write and branch, whose empty case is what makes a subtree read-only; branch name, pull request title and commit message templates each with an enforcement flag; commit signer identity and signing method; a webhook base url; and a write-only secure block for a repository token, a webhook secret and a commit signing key.

Two of those are load-bearing rather than cosmetic. An empty workflow list is the only way to express a read-only subtree, and it is currently unreachable, so every vended subtree is implicitly writable. A sync target of instance claims the whole instance rather than one folder, which collides with the single-declarative-owner rule the module was built to enforce, so it needs a decision rather than a passthrough.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The provider type is a claim input covering every type the pinned provider accepts, with the per-type url, branch, path and token-user fields expressible, and a claim whose type and per-type block disagree is rejected at admission
- [ ] #2 Sync enabled, target and interval are claim inputs, and the instance target either carries an explicit ownership decision or is refused by construction with that refusal recorded as design
- [ ] #3 The allowed-workflow list is a claim input and the empty read-only case is expressible and covered by a test
- [ ] #4 Branch, pull request and commit templates, commit signer identity and signing method, and the webhook base url are claim inputs
- [ ] #5 The repository token, webhook secret and commit signing key use the same secure-value reference route as the connection credential, and no inline credential literal is admissible
- [ ] #6 Several repositories against one stack are provable from one claim set, and the pinned provider repository CRD in the admission fixture round-trips the emitted shape for each provider type
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Root imports the pinned Repository CRD after a byte-equality provenance control.
2. Lane B widens the existing API and renderer across all provider types, repository settings, folder or folderless sync, empty workflows, and secure-value name references.
3. Root wires registries, runs the integrated gate and security review, then records exact-SHA hosted evidence.

Wave 11: widen the existing repository API and renderer across the pinned provider surface, refuse instance sync by construction, preserve the empty read-only workflow list, and use secure-value name references; the fixture pre-pass is already complete on main.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.

Wave 11 integration: provider children now reference the actual per-stack ProviderConfig named by stackRef.name, not the stale -provider suffix inherited by the preview renderer. The stack reference is immutable at admission, and an omitted repository description remains omitted rather than becoming an explicit empty value. Focused and full admission readback covered the corrected shape.

Wave 11 security correction: repository.uid and stackRef.name are immutable; provider, enterprise-server and webhook URLs reject userinfo; every secure-map alias rejects create form on create and update with an isolated weaken-admit-restore control. Corrective SECURITY review passed with no blocking findings.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.
<!-- SECTION:FINAL_SUMMARY:END -->
