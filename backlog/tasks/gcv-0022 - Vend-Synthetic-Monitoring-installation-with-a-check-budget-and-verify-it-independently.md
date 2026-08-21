---
id: GCV-0022
title: >-
  Vend Synthetic Monitoring installation with a check budget, and verify it
  independently
status: To Do
assignee: []
created_date: '2026-08-21 12:16'
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
- [ ] #1 Installation readiness is verified independently rather than trusted from the managed resource condition
- [ ] #2 Only the derived Synthetic Monitoring token is persisted; the bootstrap access policy token is not
- [ ] #3 A platform-controlled budget bounds checks, probe locations and frequency, with browser checks costed separately
- [ ] #4 Any private probe token is written to the secret store and never to composite status
- [ ] #5 The known upstream readiness defect is recorded in the README known limitations
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
