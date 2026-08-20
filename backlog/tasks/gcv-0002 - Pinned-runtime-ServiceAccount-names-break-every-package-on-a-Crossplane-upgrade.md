---
id: GCV-0002
title: >-
  Pinned runtime ServiceAccount names break every package on a Crossplane
  upgrade
status: Done
assignee: []
created_date: '2026-08-20 14:43'
updated_date: '2026-08-20 14:50'
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
- [x] #1 Neither DeploymentRuntimeConfig pins serviceAccountTemplate.metadata.name, so each package revision owns a revision-scoped ServiceAccount
- [x] #2 A note in the published guide records why the name is left unset, so a future change does not reintroduce it
- [x] #3 ./scripts/validate.sh is green and the hosted validation run is recorded with the completing SHA
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Dropped serviceAccountTemplate.metadata.name from both DeploymentRuntimeConfigs (platform/function/install.yaml, platform/provider/provider-grafana.yaml), leaving a comment at each removal site so it is not reintroduced, and added a 'For a Crossplane upgrade' subsection to the README upgrade runbook.

Verified the naming claim against the v2.4.0 source rather than inferring it: DeploymentRuntimeBuilder.ServiceAccount applies ServiceAccountWithOptionalName(b.revision.GetName()), so the default is revision-scoped and only a runtime-config name makes the object shared. In the same tag, Pre() applies the Service and both TLS secrets through applySharedRuntimeObject (which demotes a competing controller reference as it claims the object) while Post() still applies the ServiceAccount through applySA -> applyRuntimeObject. So #7714 did not cover this path and a pinned name still reproduces #7708 on v2.4.0.

Completing SHA 3e286dee6d49ea94413d3ec1e9447d9c950daebb. Local gate green (public-release scan, gofmt, go mod tidy, race tests at 89.0% coverage, vet, YAML parse, four Kustomize renders, README coverage, watch-path assertion). Hosted 'Validate public reference' run 32382028182 succeeded on that SHA.

Confirmed live on the consumer control plane that consumed this commit: both packages moved to Healthy=True within two minutes, each with a runtime Deployment named after its active revision, and every request returned to SYNCED=True READY=True. The two orphaned package-named ServiceAccounts remain, still controller-owned by the deactivated revisions, and are collected when those revisions are pruned.

Still open upstream and deliberately not fixed here: applySA should use the shared applicator, since a user-supplied ServiceAccount name is exactly the externally-managed case #7714's own note describes. Worth a follow-up issue on crossplane/crossplane.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Both runtime ServiceAccount names unpinned, so each package revision owns a revision-scoped ServiceAccount and a revision re-mint no longer deadlocks the runtime hand-off. The upgrade runbook records why the field stays unset. Fixed in 3e286de, hosted validation 32382028182, and confirmed on a live control plane that had been fully stalled by it.
<!-- SECTION:FINAL_SUMMARY:END -->
