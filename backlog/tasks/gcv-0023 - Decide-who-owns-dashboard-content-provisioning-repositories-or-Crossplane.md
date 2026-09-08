---
id: GCV-0023
title: 'Decide who owns dashboard content: provisioning repositories or Crossplane'
status: In Progress
assignee: []
created_date: '2026-08-21 12:16'
updated_date: '2026-09-08 11:53'
labels: []
dependencies: []
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
RepositoryV0alpha1 and ConnectionV0alpha1 are already present at the current pin, so Git-backed dashboard provisioning is vendable today. But a repository-provisioned dashboard and a Crossplane-managed Dashboard are two ownership models over the same object, and nothing prevents both from claiming one folder subtree.

Decide and enforce one owner per folder subtree structurally rather than by convention.

Verified inputs to the decision. The provisioning surface configures where Grafana syncs from and never the dashboard JSON, so it does not replace the current baseline Dashboard rendering. Its own dated announcement puts it in public preview, and sources conflict about general availability. Self-managed deployments additionally require two feature toggles, and no resource was found that sets Grafana feature toggles on a Cloud stack. The repository resource carries branch-workflow, pull-request-template and commit-signing sub-specs plus a secure block, and the connection resource exists so credentials are referenced rather than embedded per repository.

Also relevant: deleted-dashboard recovery is generally available, but a restored dashboard starts at version 1 with no version history, which weakens any argument for the UI as a backup and strengthens git as the source of truth.

Recommendation to evaluate, not a foregone conclusion: keep classic Dashboard rendering for baseline content, and treat provisioning repositories as an opt-in per-subtree alternative once preview status resolves.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A decision is recorded naming the owner per folder subtree and how that ownership is enforced structurally
- [ ] #2 The preview status and feature-toggle preconditions are documented as they stand at decision time
- [ ] #3 The README states which content route is the supported default and why
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 2: implement and validate provisioning repositories with structural subtree ownership exclusion and credential references; root wires and gates.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Decision, taken by the repository owner 2026-09-08: adopt provisioning repositories as a new opt-in module, and plan the move to default

RepositoryV0alpha1 and ConnectionV0alpha1 are vended as their own module. Classic Crossplane Dashboard
rendering stays the supported default for baseline content in this wave, and the README states that
provisioning repositories are the intended future default rather than a permanent alternative.

The feature-toggle precondition is struck from the decision: this vending machine supports Grafana
Cloud stacks only and will not support self-managed deployments, so the two self-managed feature
toggles are not a constraint here. Say that in the README rather than repeating the toggle caveat.

Preview status remains a real caveat and is documented as it stands at decision time, not used as a
reason to defer the module.

Ownership is enforced structurally, not by convention: a folder subtree declared as
repository-provisioned renders no Crossplane Dashboard for that subtree, and admission rejects a
request that declares both owners for one subtree. This is the same single-declarative-owner rule the
folder and dashboard ACL surfaces already follow.

Wave 1 lane H disposition, 2026-09-08: Not started and Parked because the mandatory root pre-fan-out pass did not produce a pushed seam SHA after route metadata was unavailable. Resume after GCV-0032 completes the section 5.0 pass, then spawn JUDGMENT+EXECUTION on gpt-5.6-terra at high effort with fork_turns none and the pushed pre-pass SHA. No acceptance criterion or Definition of Done item was checked.
<!-- SECTION:NOTES:END -->
