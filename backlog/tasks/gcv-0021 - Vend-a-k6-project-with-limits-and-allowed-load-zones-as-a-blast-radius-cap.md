---
id: GCV-0021
title: Vend a k6 project with limits and allowed load zones as a blast-radius cap
status: To Do
assignee: []
created_date: '2026-08-21 12:15'
labels: []
dependencies: []
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The k6 governance resources are unusually well suited to vending because they express a per-team blast-radius cap declaratively, which is rare in this product family. ProjectLimits carries vuhMaxPerMonth, vuMaxPerTest, vuBrowserMaxPerTest and durationMaxPerTest. ProjectAllowedLoadZones allow-lists load zones by identifier.

Vend the project, its limits and its allowed zones. Leave load tests and schedules to the consuming team; test content cannot be inferred from a stack request.

Two verified cautions. Installation is a bootstrap exchange: it takes a stack service account token and a user, and outputs a k6 access token plus organization, so the credential chain differs from Synthetic Monitoring which bootstraps from a Cloud access policy token instead. And the installation resource credential surface churned twice within one upstream release week, with one field added and removed again and another deprecated, so verify the shape against the actual pinned provider rather than upstream documentation.

Private load zones can only be allow-listed, never provisioned, from this provider.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Project, limits and allowed load zones are rendered; load tests and schedules are not
- [ ] #2 Limits are platform-controlled per usage class and cannot be raised by a request author
- [ ] #3 The installation bootstrap is gated on the observed stack service account token and the derived k6 credential is mirrored to the secret store, never the input token
- [ ] #4 The installation field shape is verified against the pinned provider CRD rather than upstream docs
- [ ] #5 A catalog example renders with placeholder load zone identifiers
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
