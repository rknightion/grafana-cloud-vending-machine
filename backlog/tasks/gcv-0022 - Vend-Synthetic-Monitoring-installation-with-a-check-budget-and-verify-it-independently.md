---
id: GCV-0022
title: >-
  Vend Synthetic Monitoring installation with a check budget, and verify it
  independently
status: Done
assignee: []
created_date: '2026-08-21 12:16'
updated_date: '2026-09-08 16:43'
labels: []
dependencies: []
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Vend the Synthetic Monitoring installation plus a platform-controlled check budget, leaving check definitions to the consuming team because targets cannot be inferred from a stack request.

The bootstrap is the pattern worth copying elsewhere: the installation resource takes a Cloud access policy token carrying stacks read plus metrics, logs and traces write, and outputs a narrower Synthetic-Monitoring-scoped token used for everything else. Persist only the derived token; never the input.

The critical trap is a known open upstream defect: the installation resource reports creation complete in zero seconds while Synthetic Monitoring is not actually configured on the stack. Under Crossplane this is worse than under Terraform, because the managed resource reports Ready and Synced on a no-op, and readiness is what everything else gates on. The composition must verify against the Synthetic Monitoring API rather than trusting the child resource, or downstream children will be admitted against a stack where the product was never installed.

Cost model justifies the budget: billing is executions equal to checks multiplied by probe locations multiplied by frequency, with browser executions roughly an order of magnitude dearer than API executions. Bound checks, probe count and frequency in the API.

Private probes emit a sensitive token that a self-hosted probe binary needs; treat it as a secret-store output, not a status field.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Installation readiness is verified independently rather than trusted from the managed resource condition
- [x] #2 Only the derived Synthetic Monitoring token is persisted; the bootstrap access policy token is not
- [x] #3 A platform-controlled budget bounds checks, probe locations and frequency, with browser checks costed separately
- [x] #4 Any private probe token is written to the secret store and never to composite status
- [x] #5 The known upstream readiness defect is recorded in the README known limitations
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
Installation readiness requires an independently observed disabled Check through the derived-token provider. Only derived credentials are published. Complete team-authored check sets are budgeted for count, probes, frequency and browser weighting; removed checks and stale verifiers are deletion-managed. No private Probe or probe token is created, so AC4 is satisfied by exclusion. Race tests and catalog validation passed; upstream readiness defect is documented. Completing delivery SHA bec9551c3c2abb009a4a50412b33efe47b07520c; hosted Validate 34252640140 success. Root just check passed (85.7% coverage). Signed multi-platform function digest sha256:09ff21ddf5436d0f0165ac7849d86ab4c22a6633551d91ab6aab4edc48f88652 is pinned in both locations. No live provider or deployment proof is claimed.
<!-- SECTION:FINAL_SUMMARY:END -->
