---
id: GCV-0054
title: State the cluster prerequisites the admission-policy layer introduced
status: To Do
assignee: []
created_date: '2026-09-09 08:09'
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
- [ ] #1 The installation guide states the minimum Kubernetes version the shipped manifests require, and names ValidatingAdmissionPolicy as the reason
- [ ] #2 Operator-facing documentation explains that five XRDs ship a cluster-scoped admission policy bound to their own Composition, and what an operator sees if that Composition is absent or renamed
- [ ] #3 The pinned-versions table carries the Kubernetes floor alongside the other pins, so a single table answers what a cluster must provide
- [ ] #4 The gate asserts the documented Kubernetes floor against the version the manifests actually require, so the two cannot drift apart silently
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
