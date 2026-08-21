---
id: GCV-0019
title: Vend a golden SLO per usage class
status: To Do
assignee: []
created_date: '2026-08-21 12:15'
labels: []
dependencies:
  - GCV-0018
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The SLO kind auto-generates recording rules plus fastburn, slowburn and budget-remaining alert rules from one definition, so a single vended object gives a tenant a working error budget on day one instead of an empty alerting page, and gives the platform one comparable reliability metric across every tenant.

Render it in handoff mode so tenants can tune it, using the provenance decision from the alerting bundle task rather than a separate mechanism.

Ordering: the destination datasource must exist first, since the SLO references a Prometheus-compatible datasource uid. Use the observed-resource dependency gates already established rather than a wait.

Query types available are ratio, freeform, threshold and a multi-datasource form. Prefer ratio for a golden template because it is the only one that can be parameterised safely without knowing the workload.

Avoid the alerting enrichment block for now: its assistant investigation type depends on alert enrichment, which is public preview with an explicit breaking-change warning.

Objectives and queries are workload-owned in general; what is vended here is a template per usage class, not an inferred SLO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The SLO is templated per usage class from platform-owned configuration, not inferred from a stack request
- [ ] #2 It renders only after its destination datasource is observed, using the existing gating pattern
- [ ] #3 It is rendered in handoff mode so a tenant can edit it, consistent with the alerting provenance decision
- [ ] #4 No alert enrichment block is rendered while that surface remains in preview
- [ ] #5 A catalog example renders with a ratio query and inert metric names
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
