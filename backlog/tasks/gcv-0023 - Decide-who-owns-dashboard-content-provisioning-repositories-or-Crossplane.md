---
id: GCV-0023
title: 'Decide who owns dashboard content: provisioning repositories or Crossplane'
status: To Do
assignee: []
created_date: '2026-08-21 12:16'
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
