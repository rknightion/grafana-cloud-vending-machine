---
id: GCV-0026
title: >-
  Make retention class a stack-request input, because export does not exist at
  decommission
status: Done
assignee: []
created_date: '2026-08-21 12:17'
updated_date: '2026-09-08 16:43'
labels: []
dependencies: []
ordinal: 26000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Long-lived telemetry can only be protected at stack creation, not at decommission, so retention class belongs in the request rather than in a runbook.

Verified constraints. Cloud logs export syncs Loki chunks to a customer-owned bucket but only on a rolling window of roughly seven to thirty days, so it extends retention forward rather than dumping history. No equivalent bulk export was found for metrics or traces. Logs retention itself is adjustable through a self-serve API in thirty-day multiples with a one-year maximum, but below thirty days needs a support request, and there is an open upstream issue tracking a declarative resource for it, so today it is not reconcilable. Metrics and traces retention has no self-serve API at all. Stack deletion is permanent.

The consequence is that decommissioning a stack destroys its historical metrics and traces unless a durable sink has been written to since day one. Teams typically discover this during decommissioning, when it is unfixable.

So the vendable half is the fan-out, not the retention setting: a long-term retention class provisions remote-write to a durable sink at creation. State plainly in the README that retention periods themselves cannot be reconciled and name the ticket-based routes, rather than exposing a field that silently does nothing.

Leave room for the logs retention API as an escape hatch if it becomes worth calling directly, but do not build that on this task.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A retention class in the request provisions durable fan-out at creation for the long-term class
- [x] #2 No spec field implies control over a retention period that cannot actually be reconciled
- [x] #3 The README states which retention controls are ticket-based or API-only and that metrics and traces have no bulk export
- [x] #4 The decommission runbook cross-references this as a creation-time decision
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 3: root pushes fail-closed seams; assigned lane implements owned files test-first; root audits ownership, integrates documentation and wiring, reviews and validates, verifies signed package publication, pins both references, then finalizes with exact-SHA hosted validation.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Creation-time retention class selects platform-owned durable Fleet fan-out. No retention-period request field exists. README and decommission guidance name API-only and ticket-based controls and unavailable bulk exports. Race tests and schema validation passed; actual sink receipt remains unproven. Completing delivery SHA bec9551c3c2abb009a4a50412b33efe47b07520c; hosted Validate 34252640140 success. Root just check passed (85.7% coverage). Signed multi-platform function digest sha256:09ff21ddf5436d0f0165ac7849d86ab4c22a6633551d91ab6aab4edc48f88652 is pinned in both locations. No live provider or deployment proof is claimed.
<!-- SECTION:FINAL_SUMMARY:END -->
