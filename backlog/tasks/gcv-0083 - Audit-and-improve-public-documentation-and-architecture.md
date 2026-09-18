---
id: GCV-0083
title: Audit and improve public documentation and architecture
status: In Progress
assignee:
  - '@rob'
created_date: '2026-09-18 07:58'
updated_date: '2026-09-18 07:59'
labels: []
dependencies: []
ordinal: 83000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
New users need a code-verified path through the vending module and Crossplane concepts. The current docs already cover product families but need a whole-repository accuracy audit, clearer conceptual onboarding, and diagrams that explain the actual reconciliation and ownership model without introducing source-environment details.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every public documentation page and catalog readme is checked against the current repository contracts, manifests, and renderer behavior
- [ ] #2 New-user documentation explains the request-to-reconciliation path and links progressively to installation, configuration, and API reference material
- [ ] #3 Architecture diagrams reflect only repository-verified components and ownership boundaries, are accessible, and replace any superseded Mermaid diagram
- [ ] #4 Documentation gate passes and hosted Validate public reference passes on the completed SHA
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Inventory public documentation, navigation, catalog examples, manifests, and renderer registry; record code-versus-doc drift.
2. Add a progressive newcomer path and repair inaccurate or missing explanations using repository and current Crossplane documentation evidence.
3. Replace the Mermaid architecture sketch with accessible standalone diagrams derived from the deployed request and reconciliation path.
4. Run the mandated de-AI and technical-clarity pass after all prose is drafted, then validate locally, review the final diff, commit, push, and verify the hosted gate on the exact SHA.
<!-- SECTION:PLAN:END -->
