---
id: GCV-0012
title: Expose observability product activation on the stack request
status: To Do
assignee: []
created_date: '2026-08-21 12:14'
labels: []
dependencies: []
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Three provider kinds activate Grafana Cloud observability products per stack: AppO11yConfigV1alpha1, DBO11yConfigV1alpha1 and K8sO11yConfigV1alpha1, all in the cloud family. Each is a singleton with a fixed metadata uid of global and a spec carrying an enabled boolean. They need no bootstrap and no credential beyond the per-stack ProviderConfig this repository already vends.

Add a products object to the stack request spec with three booleans, defaulting off, rendering one singleton per enabled product.

State plainly in the README that these are activation toggles, not configuration. The configurable surface for Kubernetes Monitoring lives in Helm chart values and for Application and Database Observability in onboarding flows, none of which is reachable from this provider. An API field that implies otherwise is worse than an absent one.

Verified present in the pinned provider, so this needs no pin bump.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 spec.products carries applicationObservability, kubernetesObservability and databaseObservability booleans, each defaulting false
- [ ] #2 Each enabled product renders its singleton with the fixed global external name and no others
- [ ] #3 Disabling a product after enabling it is handled explicitly and the behaviour is documented
- [ ] #4 The README states these are activation toggles and names where the real configuration lives
- [ ] #5 A catalog example enables at least one product and renders
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
