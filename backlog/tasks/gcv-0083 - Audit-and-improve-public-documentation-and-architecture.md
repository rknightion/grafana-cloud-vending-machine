---
id: GCV-0083
title: Audit and improve public documentation and architecture
status: Done
assignee:
  - '@rob'
created_date: '2026-09-18 07:58'
updated_date: '2026-09-18 08:17'
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
- [x] #1 Every public documentation page and catalog readme is checked against the current repository contracts, manifests, and renderer behavior
- [x] #2 New-user documentation explains the request-to-reconciliation path and links progressively to installation, configuration, and API reference material
- [x] #3 Architecture diagrams reflect only repository-verified components and ownership boundaries, are accessible, and replace any superseded Mermaid diagram
- [x] #4 Documentation gate passes and hosted Validate public reference passes on the completed SHA
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Inventory public documentation, navigation, catalog examples, manifests, and renderer registry; record code-versus-doc drift.
2. Add a progressive newcomer path and repair inaccurate or missing explanations using repository and current Crossplane documentation evidence.
3. Replace the Mermaid architecture sketch with accessible standalone diagrams derived from the deployed request and reconciliation path.
4. Run the mandated de-AI and technical-clarity pass after all prose is drafted, then validate locally, review the final diff, commit, push, and verify the hosted gate on the exact SHA.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Audit evidence: reviewed the public documentation pages, catalog README inventory, XRD and Composition manifests, renderer registry, validation gate, and current Crossplane documentation. Local Markdown links resolved for 45 files. The only material gap was newcomer orientation and the Mermaid-only system view; no source-versus-public-documentation contradiction was found.

Validation: just check passed locally. Hosted Validate public reference run 35323087134 and ci-success passed for implementation SHA cb663efc86e82242df9b2c80783d977ee13329d5.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added a source-grounded Crossplane concepts page, linked it through the documentation map and navigation, and replaced the Mermaid architecture sketch with accessible ownership and request-lifecycle diagrams. Completed implementation SHA cb663efc86e82242df9b2c80783d977ee13329d5; hosted validation run 35323087134 passed.
<!-- SECTION:FINAL_SUMMARY:END -->
