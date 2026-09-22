---
id: GCV-0090
title: >-
  Renovate cannot update the pinned-versions table, so every version-bearing
  component bump red-lights the gate
status: To Do
assignee: []
created_date: '2026-09-22 15:56'
labels:
  - needs-triage
  - ci
dependencies: []
references:
  - 'https://github.com/rknightion/grafana-cloud-vending-machine/pull/46'
type: bug
ordinal: 90000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The pinned-versions table in docs/installation.md names an exact source locator per row, and scripts/validate.sh requires that literal to exist in the named platform/ or deploy/ file. Renovate updates the manifest and cannot update the markdown, so the two disagree from the moment the bump lands and the gate aborts.

Observed on PR 46, the cosign image tag bump v3.1.2 to v3.1.3. Renovate changed platform/function/install.yaml and platform/provider/provider-grafana.yaml. docs/installation.md still carries the locator cosign/cosign:v3.1.2, so the Validate reference job aborted with 'docs/installation.md: pinned component Cosign verification image locator is absent from platform/function/install.yaml'. That PR has failed hosted Validate four times across four days with no intervening change, because nothing in the automation can close the gap.

The drift check itself is correct and is the control GCV-0057 shipped. The defect is that no automation can satisfy it, so the class is 'every Renovate bump of a component whose version appears in the table is red until a human edits prose'. The affected rows are any whose locator embeds a version rather than only a digest; a digest-only locator is already covered because the separate digest check reads both sides by discovery.

Two candidate repairs, and the choice is the first thing to settle. Either add a Renovate custom manager over docs/installation.md so the table row moves in the same PR as the manifest, which keeps the locator literal and adds a second place the version is pinned; or make the locator version-free so the table names the component and source path while the exact version is discovered from the manifest at validation time, which removes the drift surface instead of automating it. The second is smaller and removes a class rather than covering it, but it weakens what the table publicly states.

Verify against a real bump rather than by reasoning: the proof is a Renovate PR going green without a hand edit, or the validator rejecting a genuinely drifted table.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The root cause is confirmed by reproducing a component version bump in the manifest alone and observing the exact validator abort, not inferred from PR 46's log
- [ ] #2 The chosen repair is recorded with the alternative considered, including what the pinned-versions table still publicly guarantees after the change
- [ ] #3 Every pinned-versions row is audited for a version-bearing locator, so the fix covers the class rather than the cosign row
- [ ] #4 A manifest-only version bump passes the gate with no markdown edit, proven by a commit that changes only the manifest
- [ ] #5 A genuinely drifted table still fails the gate by path and component, proven by a negative control
- [ ] #6 PR 46 reaches a green hosted Validate run without a hand-written documentation edit, or the reason it still cannot is recorded
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
