---
id: GCV-0079
title: >-
  Provision datasources, a folder and Git Sync into a stack the platform does
  not own, driven by an external project CR
status: To Do
assignee: []
created_date: '2026-09-18 07:44'
labels:
  - needs-triage
  - integration
dependencies: []
references:
  - 'https://github.com/rknightion/grafana-cloud-vending-machine/issues/40'
type: feature
ordinal: 79000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Raised as GitHub issue #40, the 'diet' half of a two-part request from a Kubernetes-platform operator team whose tenancy model creates a project CR per environment. When a new project appears they want a small, fixed set of Grafana content provisioned into a Grafana Cloud stack that already exists and that this platform did not vend, without adopting the whole stack. Full stack vending from the same trigger is the other half and is tracked separately.

WHAT THEY ASKED FOR, and each item's distance from what exists today.

1. A FIXED CONTENT SET into an existing stack: a couple of datasources, one folder, a Git Sync connection and the dashboards that connection syncs. Every one of these kinds is already vendable and GCV-0061 already settled minting credentials against, and partially adopting, a stack this platform did not vend. So the composition side is mostly assembly rather than new API surface.

2. THE DATASOURCES POINT AT A DIFFERENT STACK. A separate central stack exposes the metrics and logs backends; the per-project stack gets datasources that read from it. That makes the interesting resource a credential in the central stack, scoped down, handed to a datasource in the project stack. The central stack is only partially adopted, and the platform mints access policy tokens there carrying read and query scopes for metrics and logs with the per-project label restriction attached to the policy. This is the first vended surface where one claim causes a write in two different stacks, and the cross-stack direction is the load-bearing design question: which composite owns the central-stack credential, how its lifetime and rotation relate to the project stack's, and what happens to the project datasource when the central credential is revoked.

3. THE TRIGGER IS SOMEONE ELSE'S CR. They want an annotation on a project CR, in a cluster this platform runs beside, to produce the claim. Nothing in this repository reads a third-party CR: the ApplicationSet watches only top-level enabled/*, deliberately, and the gate asserts it. So the trigger is a separate component - a generator, a small controller, or an ApplicationSet generator against the project CRs - and NOT a widening of the watch path. Settle which, and where it lives, before writing composition code. Both halves of this request need it, so whichever half is built first owns the decision.

CARRY FORWARD from the Git Sync work: the vended Git Sync connection reaches the vendor only in the credential create form, because the secure-value reference form is refused. Any Git Sync in this flow inherits that, and the refused shape must not be reintroduced. See GCV-0074.

The requesting team's own identifiers, cluster names and stack slugs are deliberately absent: write the shape, never the instance.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The set of resources provisioned per project is explicit and bounded, and provisioning into a stack the platform did not vend requires no full adoption of that stack
- [ ] #2 A datasource in one stack can read from a backend exposed by a second stack, using a credential the platform mints in that second stack, with the per-project restriction expressed on the credential rather than on the datasource
- [ ] #3 Ownership, lifetime and revocation of the cross-stack credential are recorded as a decision, including what the project-stack datasource does when it is revoked
- [ ] #4 The mechanism that turns an external project CR into a claim is chosen and recorded, and it does not widen the ApplicationSet watch path beyond enabled/*
- [ ] #5 A project whose claim cannot be satisfied fails loudly rather than reconciling a partial content set, and an operator can tell which of the two stacks refused
- [ ] #6 The Git Sync connection in this flow uses the credential form the vendor accepts, proven by a renderer test rather than by prose
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
