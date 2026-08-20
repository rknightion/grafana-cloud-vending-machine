---
id: GCV-0005
title: 'Onboard Renovate, including the managers that are inert by default'
status: To Do
assignee: []
created_date: '2026-08-20 16:10'
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
- [ ] #1 renovate.json exists on main and names files for the crossplane and kubernetes managers
- [ ] #2 Provider package updates require dependency dashboard approval and do not automerge
- [ ] #3 The self-published function package is excluded from dependency updates
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
