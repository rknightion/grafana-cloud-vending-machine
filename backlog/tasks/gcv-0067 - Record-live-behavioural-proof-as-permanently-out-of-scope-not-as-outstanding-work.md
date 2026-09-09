---
id: GCV-0067
title: >-
  Record live behavioural proof as permanently out of scope, not as outstanding
  work
status: In Progress
assignee:
  - '@claude'
created_date: '2026-09-09 21:28'
updated_date: '2026-09-09 22:42'
labels: []
dependencies: []
ordinal: 67000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every wave report closes by listing live minting, adoption, migration and reconciliation as unproven. That phrasing is accurate but it reads as a backlog item, and it invites each new run to treat obtaining that proof as available work. It is not: the repository first constraint is that it never touches a live Grafana Cloud stack, a live cluster or any source environment, so this proof cannot be produced here by construction and no future wave can close it. The boundary needs to be written down as settled with its reasoning, alongside what does stand in for it - pinned upstream source, ephemeral local API-server admission, and provider CRD readback - so a reader can tell a deliberate limit from a gap. Wave 9 also showed the constraint is easier to cross than assumed: a client-side kubectl validation ran against whatever kubeconfig context happened to be current, and read contact against a real cluster could not afterwards be ruled out.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The settled boundary is recorded where a future run will read it before designing a wave, stating that live behavioural proof is out of scope by construction and why
- [ ] #2 The record names what evidence classes stand in for live proof and what each of them does and does not establish
- [ ] #3 The published documentation states the same boundary in operator-facing terms, so a consumer is not left believing the reference was exercised against a live stack
- [ ] #4 The constraint states that any local cluster tooling invocation runs against an explicitly empty or ephemeral kubeconfig, so an ambient context cannot be contacted by default
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add the settled boundary to the repository wave operating model, where a run reads it before designing a wave, including the ambient-kubeconfig constraint wave 9 crossed.
2. Add the operator-facing statement to the documentation landing page, so a consumer cannot infer live validation from a green gate.
3. Name the evidence classes that stand in for live proof and state the limit of each.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Recorded in two places, for two audiences.

The wave operating model document gains a section stating that live behavioural proof is out of scope by construction, that no wave can close it and none should plan to, and that reports should say "not exercised, and not exercisable here" rather than "unproven", because the latter reads as a backlog item and invites each run to re-decide it. It names the four evidence classes that stand in for live proof with the limit of each, and uses wave 9 R17 as the worked example of why provider CRD readback against a self-written fixture cannot testify about provider behaviour: every gate was green while the comparator expected a slash and the pinned provider joins with a colon.

The same section carries the ambient-kubeconfig constraint. Wave 9 ran `kubectl apply --dry-run=client --validate=true` with no context scoping. Checked while reviewing that report: this machine has exactly one kubeconfig context, it is current, it is a real cluster, and the discovery cache is populated, so read contact could not afterwards be ruled out. Nothing was mutated and nothing leaked, but the constraint stopped being provable, which is the whole value of having it. Cluster tooling now runs against an explicitly empty or ephemeral kubeconfig, and a client-side dry run is recorded as not being evidence of zero network contact.

The documentation landing page gains an operator-facing section with the same boundary in consumer terms and a table of what each evidence class establishes and does not. It ends by directing the reader to rehearse every migration procedure against a disposable stack, which is what the boundary means in practice for someone adopting this.
<!-- SECTION:NOTES:END -->
