---
id: GCV-0024
title: Model a multi-stack promotion ladder as a first-class API
status: In Progress
assignee: []
created_date: '2026-08-21 12:17'
updated_date: '2026-09-08 15:15'
labels: []
dependencies:
  - GCV-0023
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Model the environment ladder as the vended object: one composite producing a linked set of stacks with a defined content-promotion direction, provisioning repositories per rung pointed at different branches, and inter-rung drift surfaced as composite status. Teams currently hand-roll this and get it wrong.

Context on topology, which matters for how this is framed. Grafana guidance currently recommends a single production stack with isolation from teams, folders, RBAC, datasource permissions and LBAC, endorsing multiple stacks for a dev and staging ladder and for complete departmental isolation. Real enterprise customers do choose multi-stack for genuine reasons, so this repository supports both topologies deliberately rather than treating either as a deviation. The existing stack request is shaped for stack-per-tenant and the team-access and content-access APIs are shaped for slices of a shared stack; both are intended. Document that explicitly rather than leaving it implicit.

Hard constraints, all verified. The region slug forces replacement, so region must stay immutable and a change must never silently recreate a stack. Stack caps are one on the free tier and three on self-service paid plans, beyond which the limit is contractual rather than a technical setting, so a many-stack ladder presumes a negotiated limit. Cloud stacks are single-organisation, so multi-org isolation is not available. Multi-stack datasources are limited to the same region and were capped at ten stacks in preview.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The README states that both shared-stack and stack-per-tenant topologies are supported deliberately, with the trade-offs
- [ ] #2 Region immutability is enforced such that no ladder operation can trigger stack replacement
- [ ] #3 The promotion direction is explicit in the API rather than implied by naming
- [ ] #4 Inter-rung drift is surfaced as composite status
- [ ] #5 The stack-cap and single-organisation constraints are documented as preconditions
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 3: root pushes fail-closed seams; assigned lane implements owned files test-first; root audits ownership, integrates documentation and wiring, reviews and validates, verifies signed package publication, pins both references, then finalizes with exact-SHA hosted validation.
<!-- SECTION:PLAN:END -->
