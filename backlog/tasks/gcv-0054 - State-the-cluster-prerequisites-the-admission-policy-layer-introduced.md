---
id: GCV-0054
title: State the cluster prerequisites the admission-policy layer introduced
status: Done
assignee:
  - '@codex-wave7'
created_date: '2026-09-09 08:09'
updated_date: '2026-09-09 10:53'
labels: []
dependencies: []
ordinal: 54000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Wave 6 added ValidatingAdmissionPolicy and binding documents to five XRD files (cloud-integrations, k6, ml, pdc, service-accounts). They use `admissionregistration.k8s.io/v1`, which is GA only from Kubernetes 1.30, and every binding sets `parameterNotFoundAction: Deny` with a `paramRef` naming that file@s own Composition by name. No document in `docs/`, `deploy/` or `README.md` states a minimum Kubernetes version, and the operator-facing installation guide does not mention that an admission layer exists at all. On a cluster below 1.30 the Argo sync of `platform/` fails outright; if a Composition is renamed or not installed, every request to that surface is denied with a message that names a policy the operator has never read about. `docs/reference/request-schema.md` does describe the coupling, but a reader planning a cluster does not start there.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The installation guide states the minimum Kubernetes version the shipped manifests require, and names ValidatingAdmissionPolicy as the reason
- [x] #2 Operator-facing documentation explains that five XRDs ship a cluster-scoped admission policy bound to their own Composition, and what an operator sees if that Composition is absent or renamed
- [x] #3 The pinned-versions table carries the Kubernetes floor alongside the other pins, so a single table answers what a cluster must provide
- [x] #4 The gate asserts the documented Kubernetes floor against the version the manifests actually require, so the two cannot drift apart silently
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Derive the Kubernetes floor from all shipped platform API versions; document the admission-policy and named-Composition failure modes; after GCV-0057 lands, add a drift assertion and prove it with a weaken/reject/restore control.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Documented Kubernetes 1.30+ from the discovered platform API inventory, named all five cluster-scoped admission bindings and their missing-Composition denial behavior, and added a drift assertion. Negative control weakened the table to 1.29 and the full gate rejected docs/installation.md against the derived 1.30 floor; restored just check passed with 84.3% coverage and zero skips.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Documented Kubernetes 1.30+ and all five cluster-scoped admission bindings, including the named-Composition failure mode. The gate derives and checks the floor; weaken to 1.29 was rejected before restoration. Completing SHA b998937b1af2ccaf852bc677602b813dcce850d4; hosted Validate public reference run 34335459253 succeeded; local just check passed at 84.3% with zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
