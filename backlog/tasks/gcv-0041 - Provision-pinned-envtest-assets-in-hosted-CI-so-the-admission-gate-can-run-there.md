---
id: GCV-0041
title: >-
  Provision pinned envtest assets in hosted CI so the admission gate can run
  there
status: To Do
assignee: []
created_date: '2026-09-08 19:53'
labels: []
dependencies:
  - GCV-0034
priority: high
type: bug
ordinal: 41000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The admission harness is deliberately part of just check rather than a heavy ci leg, and check is what the hosted Validate public reference workflow runs byte for byte. That workflow installs ripgrep, Go and just and nothing else. KUBEBUILDER_ASSETS is asserted only in just setup, which CI never calls. The harness is correctly forbidden from skipping when its binaries are absent, so the moment the wiring pass adds it to scripts/validate.sh the hosted run goes red and GCV-0034 cannot satisfy its second Definition of Done item. Nothing in waves 1 to 4 covered this because the harness was never wired in. controller-tools publishes the binaries as release assets on the exact tag the justfile renovate annotation already tracks, each with a published sha512 sidecar, so the provisioning is pinned and checksum-verifiable without hand-transcribing anything.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The hosted Validate public reference workflow provisions kube-apiserver and etcd at the version the justfile pins, and exports KUBEBUILDER_ASSETS before just check runs
- [ ] #2 The archive is verified against its own published sha512 sidecar from the same release, with no checksum transcribed into the repository by hand
- [ ] #3 The pinned version has exactly one source of truth: a version bump in the justfile changes what CI installs, with no second place to update
- [ ] #4 A deliberately wrong checksum or a missing asset fails the workflow step loudly, proven by a negative control, rather than continuing to a gate that skips
- [ ] #5 just setup and the hosted workflow agree on the same assets, so a clean local checkout and a clean CI run reach the same gate result
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
