---
id: GCV-0079
title: >-
  Provision datasources, a folder and Git Sync into a stack the platform does
  not own, driven by an external project CR
status: Done
assignee:
  - '@codex'
created_date: '2026-09-18 07:44'
updated_date: '2026-09-24 10:25'
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
- [x] #1 The set of resources provisioned per project is explicit and bounded, and provisioning into a stack the platform did not vend requires no full adoption of that stack
- [x] #2 A datasource in one stack can read from a backend exposed by a second stack, using a credential the platform mints in that second stack, with the per-project restriction expressed on the credential rather than on the datasource
- [x] #3 Ownership, lifetime and revocation of the cross-stack credential are recorded as a decision, including what the project-stack datasource does when it is revoked
- [x] #4 The mechanism that turns an external project CR into a claim is chosen and recorded, and it does not widen the ApplicationSet watch path beyond enabled/*
- [x] #5 A project whose claim cannot be satisfied fails loudly rather than reconciling a partial content set, and an operator can tell which of the two stacks refused
- [x] #6 The Git Sync connection in this flow uses the credential form the vendor accepts, proven by a renderer test rather than by prose
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 16 lane D records the credential lifecycle decision before implementing the new API, renderer, tests, and documentation; root owns registry wiring, review, publication, and final gate.
<!-- SECTION:PLAN:END -->

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

## Owner decision 2026-09-22 after wave 16: one specialist rescue, and the AC5 failure contract

**Attempt accounting.** Two of the four Codex lane attempts are consumed: wave 16 lane D (attempt 1) and the wave 16 root rescue (attempt 2, rejected on rereview with the same signature as the initial review). Wave 17 may take exactly one more, attempt 3, as a specialist rescue. It runs on gpt-6-astra at effort high. That is an explicit operator exception to the Codex profile's rule against launching Astra/high, and it covers this one attempt only: not a retry after it, and not another lane. If attempt 3 is rejected the task parks. Attempt 4 would need a new owner decision.

**Correction to the wave 16 final summary.** 'Resume only with a new owner-authorized attempt' was a condition the wave 16 root added, not one the protocol or its goal imposed. The standing stop rule forbade the root from taking a second repair itself. It did not stop a specialist rescue within the budget, and it did not stop the other lanes.

**AC5 failure contract: refusal only changes status.** The renderer always emits the complete deterministic child set, built from the claim, the profile and the observed provider-assigned IDs. A refusal by either stack is never a renderer error. It sets that side's status.projectContent.<side> to Refused and the composite Ready=False, and adds one warning that carries no provider text. This follows the GCV-0087 credential-health precedent: children are kept and readiness goes false. Fatal is kept for a request that genuinely cannot be rendered, and only after pinned Crossplane core source, read at the exact pinned revision, proves that a Fatal result skips both apply and garbage collection. The rejected alternatives: Fatal-always with the side named in the message, because Ready can stay stale-True and the per-side status is never written; and re-emitting observed children verbatim, because that endorses drift.

**Proof boundary for AC5.** A real RunFunction response with an empty incoming desired state and all 13 children observed must carry all 13 authored identities and specs for central refusal, project refusal and dependency disappearance. A test that pre-populates req.Desired proves nothing about preservation.

Wave 17 attempt 3 produced local candidate c7877b26caa0692586d8599653e38628e4f2e447 and passed the integrated local gate, but independent security review REJECTED it. Lane H reproduced the E3 signature: removing an unused project observer ID or refusing that observer returned zero desired children plus Fatal although all 13 children remained renderable from the claim/profile and observed dependencies. Lane H also proved a mismatched access-policy external-name could be copied into desired state while the token retained the original policy ID and both sides reported Ready. Review artifact codex/wave17/lane-h-review.md SHA-256 dba2290ee1ec2b4ab1efc4cc44a625735cf7dc2fa7320421ba0c84b1b118212d. No candidate was pushed or published; task remains Parked and no acceptance criterion changed.

2026-09-24 owner authorizes final attempt 4 of 4 against H1 and H2, followed by independent security review. Rejection parks; no fifth attempt without new owner decision.

Loop 18 final attempt 4 is on rebased origin/main, with H1/H2 failing-first RunFunction controls and independent security review H.

Loop 18 final attempt 4 repaired H1 missing/refused project observer and H2 policy identity mismatch with real RunFunction empty-incoming-desired tests. H independently approved exact candidate 1682086f; integrated reviewed source blobs remained identical. Full integrated just check passed at ba1475d with 85.5 percent coverage; hosted Validate 35984785494 and signed package Publish 35984785527 succeeded. Pin 886887d hosted Validate 35986358148 succeeded. Renderer/admission proof only; live cross-stack query and provider reconciliation are not exercised here.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 16 parked before landing. Lane D implemented the new bounded cross-stack surface and root attempted the AC5 response integration, but the permitted rereview reproduced the same major preservation failure as the initial review: with 13 observed children and empty incoming desired state, central provider refusal returned desired=0. The same rereview also found TestProviderFamilyDocumentation unable to resolve the local renderer dispatch. No candidate was committed, published, deployed, or exercised live. Resume only with a new owner-authorized attempt that proves a fresh single-step pipeline response retains all 13 children on either refusal and dependency disappearance, fixes provider-family documentation coverage, then receives a fresh adversarial review. Do not reuse the prepopulated-desired test as preservation proof.

Wave 17 specialist attempt 3 was rejected by independent security review on the repeated child-withdrawal/Fatal signature and a new policy identity mismatch. The local candidate was not landed or published. Resume only from the two exact Lane H counterexamples; the root is not authorized to repair this code.

Shipped bounded cross-stack project content. Status-only refusals retain all 13 authored children; policy identities are validated before desired state is written. Independent security review approved; released API files remained unchanged. Completing source SHA ba1475df2ea802a0bbec678986fba80807d5f603, hosted Validate 35984785494; signed publish 35984785527; delivered by pin SHA 886887d0315a2a6ae5182732f184cc873a5d1fa2, hosted Validate 35986358148. Live provider behavior is not exercised here.
<!-- SECTION:FINAL_SUMMARY:END -->
