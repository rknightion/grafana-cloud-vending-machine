---
id: GCV-0071
title: Complete the Git Sync repository surface the vending API pins to one shape
status: To Do
assignee: []
created_date: '2026-09-12 12:44'
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
