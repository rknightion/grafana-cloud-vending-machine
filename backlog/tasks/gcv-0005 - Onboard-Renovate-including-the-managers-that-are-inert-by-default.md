---
id: GCV-0005
title: 'Onboard Renovate, including the managers that are inert by default'
status: Done
assignee: []
created_date: '2026-08-20 16:10'
updated_date: '2026-08-20 16:20'
labels:
  - ci
dependencies: []
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Renovate autodiscovers this repository and opened its onboarding pull request on 2026-08-04, which was never merged, so nothing was ever tracked.

Merging the onboarding config alone would not have been enough. The crossplane and kubernetes managers ship no default managerFilePatterns -- there is no common naming convention for those YAML files, so Renovate declines to scan every *.yaml on the chance one is a package definition -- and both stay inert until a repository names its files. Without patterns Renovate would have watched the Go module and the workflow Actions while both package digests and the cosign verifier stayed invisible.

Fleet policy (SHA pinning, immutable Action digests, cooldown, automerge) is inherited from the self-hosted admin config via autodiscover, so only repo-specific configuration belongs here.

Provider bumps must not open a pull request unprompted: the digest is pinned in two places and the gate asserts they match, so an automatic edit of spec.package alone lands red, and the upgrade runbook requires changelog review, generated CRD schema diffs for every activated kind, and a disposable stack. Dependency dashboard approval gives the visibility without the noise. The function package is published by this repository and advances through its own workflow, so it is excluded outright.

Validate config changes against the version the fleet bot actually runs. The renovate package that npx resolves by default is far older and rejects managerFilePatterns as an unknown option, because it predates the rename from fileMatch.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 renovate.json exists on main and names files for the crossplane and kubernetes managers
- [x] #2 Provider package updates require dependency dashboard approval and do not automerge
- [x] #3 The self-published function package is excluded from dependency updates
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
renovate.json names platform/function/install.yaml and platform/provider/provider-grafana.yaml for both the crossplane and kubernetes managers, which is what makes the two package digests and the cosign verifier visible at all. Provider updates carry dependencyDashboardApproval with automerge disabled; the self-published function package is disabled outright.

Validated against renovate 44.31.0, the version the fleet bot runs per the admin config's docker run: 'Config validated successfully'. Worth knowing for next time: the renovate package npx resolves by default is 37.440.7, which predates the fileMatch to managerFilePatterns rename and rejects both manager blocks as invalid options. That failure means the validator is stale, not the config.

Committing config to the default branch is what onboards the repository; the onboarding pull request open since 2026-08-04 becomes redundant and Renovate closes it on its next run.

Completing SHA 9a2b73d, hosted validation run 32390912362.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Repository onboarded with the crossplane and kubernetes managers pointed at the two package files, so package digests and the signature verifier are tracked; provider bumps wait for dashboard approval and the self-published function package is excluded. Validated against the fleet bot's own Renovate version.
<!-- SECTION:FINAL_SUMMARY:END -->
