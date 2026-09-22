---
id: GCV-0080
title: >-
  Vend a whole Grafana Cloud stack in response to an external project CR
  appearing
status: To Do
assignee: []
created_date: '2026-09-18 07:44'
updated_date: '2026-09-22 15:57'
labels:
  - needs-triage
  - integration
dependencies: []
references:
  - 'https://github.com/rknightion/grafana-cloud-vending-machine/issues/39'
type: feature
ordinal: 80000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Raised as GitHub issue #39, the full-vend half of a two-part request from a Kubernetes-platform operator team. Their tenancy model creates a project CR per environment, and they want a new project appearing to produce a whole Grafana Cloud stack, from a template, with no human writing a request. The narrower request for content into an already-existing stack is tracked separately as GCV-0079.

WHAT IS ALREADY THERE, and what is not. The stack itself is the oldest and most exercised surface in this platform: GrafanaCloudStackRequest vends a stack, its administrator and telemetry credential chains, its per-stack ProviderConfig and its baseline content, and GCV-0032 settled multi-organization vending. So this issue adds almost nothing to the composition function. What it adds is a TRIGGER and a TEMPLATE, and both sit outside the composition layer.

THE TRIGGER. Nothing in this repository reads a third-party CR, from any cluster. The ApplicationSet watches exactly top-level enabled/* and scripts/validate.sh asserts that equality, precisely so inert examples cannot become live requests. Widening that glob is the wrong answer and the gate will refuse it. The right answer is a separate component that observes project CRs and writes a claim - a controller, a generator, or an ApplicationSet generator pointed at the project CRs - and the choice of which, and whether it lives in this repository at all, is the first decision. GCV-0079 needs the same component, so whichever is built first owns that decision and the other consumes it.

THE TEMPLATE. A claim generated from a CR has to get its values from somewhere: which fields come from the project CR, which are fixed by the platform, and what happens to a project whose CR carries a value the request schema refuses. The usage classification is a constrained enum, the organization field is required and immutable, and the output identity is derived from both, so a generated claim can be refused on rules a hand-written one would never hit. The generated claim must be reviewable before it reconciles, or an operator cannot tell a templating mistake from a vendor refusal.

TWO CONSEQUENCES worth naming now. Automatic vending removes the human step that today bounds how many stacks exist, so the template needs a cap or an approval seam, not just a mapping. And stack decommission is non-destructive by design here, so a project CR disappearing must not be wired to stack deletion without the owner-authorised decommission path.

The requesting team's own identifiers, cluster names and stack slugs are deliberately absent: write the shape, never the instance.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The mechanism that turns an external project CR into a stack request is chosen and recorded, with the alternatives considered, and it does not widen the ApplicationSet watch path beyond enabled/*
- [ ] #2 The mapping from project CR to request fields is explicit, states which values the platform fixes, and states what happens to a project whose CR value the request schema refuses
- [ ] #3 A generated request is reviewable before it reconciles, and a templating error is distinguishable from a vendor refusal without reading provider logs
- [ ] #4 The blast radius of automatic vending is bounded by an explicit cap or approval seam, not only by the rate projects are created
- [ ] #5 A project CR disappearing does not delete a vended stack except through the existing owner-authorised decommission path
- [ ] #6 Whether this component belongs in this repository or in the consuming platform is recorded as a decision, given the public-reference constraint
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Owner decision 2026-09-22: the external-CR trigger belongs to the consuming platform

Settled by the repository owner, and it is the shared seam GCV-0079 and GCV-0080 both said the first-built half would own. Recorded once here and on the sibling task.

**The trigger is not built in this repository.** A project CR appearing in someone else's cluster is turned into a claim by that platform's own automation. This repository ships the contract that automation consumes: the claim schema, an explicit field mapping from a project-CR shape, the approval seam, the cap, and a worked inert example. Nothing new runs here and no watch path widens.

The reasoning, because the alternatives are not obviously worse until stated:

- A controller here would have to encode the requesting team's CRD shape, which is their tenancy model, so a portable public reference would be publishing one consumer's internal schema. It also means a new runtime component with its own image, publish workflow, signature, digest pin and RBAC, and a second serialized publishing bottleneck beside the function package.
- An Argo generator route is a Plugin generator, which is an HTTP service, so it is a runtime component under a different name. Its claim never lands in Git, so there is no reviewable artefact and GCV-0080 AC3 has to be built separately rather than falling out of the design.
- With a pull request into enabled/ as the seam, GCV-0080 AC3 and AC4 are satisfied by construction: the generated request is reviewable before it reconciles, and the human step is the cap. AC5 inverts too, because nothing deletes a stack unless a person does it, whereas a controller owning claims propagates a disappearing project CR by default.

**Gap found while settling this, and it is not covered by an existing control.** scripts/validate.sh asserts the requests ApplicationSet generator's directories equals exactly enabled/*. That catches a widened glob and a second generator block added to that ApplicationSet. It does not catch a wholly separate ApplicationSet introducing a second input source, which is exactly the shape the rejected generator route would have taken. The assertion should cover the class, not the one object.

**What closing these tasks requires**, per the owner: ship what this repository can, which is the documented claim contract, the field mapping, the usage guidance and the recorded boundary. The trigger's absence is the answer to GCV-0080 AC6, not a deferral of it.
<!-- SECTION:NOTES:END -->
