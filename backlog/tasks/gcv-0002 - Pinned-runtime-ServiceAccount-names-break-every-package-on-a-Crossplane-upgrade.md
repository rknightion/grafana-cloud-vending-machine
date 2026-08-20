---
id: GCV-0002
title: >-
  Pinned runtime ServiceAccount names break every package on a Crossplane
  upgrade
status: To Do
assignee: []
created_date: '2026-08-20 14:43'
labels:
  - crossplane
  - platform
dependencies: []
type: bug
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Both DeploymentRuntimeConfigs in platform/ pin an explicit runtime ServiceAccount name (one for the composition function, one for the provider). Crossplane's manifest builder only applies that name when the runtime config supplies one; the default is the *revision* name, so a pinned name is the one configuration that makes the ServiceAccount a single object shared by every revision of a package.

Crossplane v2.4.0 changed the package revision id from the raw image digest to a hash of digest and generation, so upgrading from v2.3.x re-mints a revision for every installed package. crossplane/crossplane#7708 covers the resulting failure: package runtime objects moved to server-side apply in #7563, ownerReferences is a merge-keyed list, so the incoming revision's controller reference merges alongside the outgoing revision's instead of replacing it, and the API server rejects two controller references on one object.

#7714 fixed that for the Service and the TLS secrets by way of a new applySharedRuntimeObject, which demotes a competing controller reference as it claims the object. The ServiceAccount was left on the plain applicator (applySA -> applyRuntimeObject) because with default naming it is revision-scoped and never shared. A pinned name puts it back in the shared class without the shared applicator, so on upgrade the new revision's post-establish hook fails with 'Only one reference can have Controller set to true', no runtime Deployment is ever created, and the composition pipeline fails closed with DeadlineExceeded / no children to pick from. Every request on the control plane stops reconciling, deletions included.

Observed on a consumer control plane upgrading from chart 2.3.4 to 2.4.0: both packages Installed=True Healthy=False, zero runtime pods, RuntimeHealthy=False on both active revisions, and the ServiceAccount still controller-owned by the deactivated revision.

Nothing in this repository references either ServiceAccount by name -- no RBAC subject, no pull-secret wiring, no annotation -- so the pinned names buy stable cosmetics and cost an upgrade outage. Drop them.

Worth reporting upstream separately: applySA should use the shared applicator, since a user-supplied name is exactly the case the #7714 note about externally-managed ServiceAccounts describes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Neither DeploymentRuntimeConfig pins serviceAccountTemplate.metadata.name, so each package revision owns a revision-scoped ServiceAccount
- [ ] #2 A note in the published guide records why the name is left unset, so a future change does not reintroduce it
- [ ] #3 ./scripts/validate.sh is green and the hosted validation run is recorded with the completing SHA
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
