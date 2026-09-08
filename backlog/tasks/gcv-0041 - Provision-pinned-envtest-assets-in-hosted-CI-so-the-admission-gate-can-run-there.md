---
id: GCV-0041
title: >-
  Provision pinned envtest assets in hosted CI so the admission gate can run
  there
status: Done
assignee: []
created_date: '2026-09-08 19:53'
updated_date: '2026-09-08 21:01'
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
- [x] #1 The hosted Validate public reference workflow provisions kube-apiserver and etcd at the version the justfile pins, and exports KUBEBUILDER_ASSETS before just check runs
- [x] #2 The archive is verified against its own published sha512 sidecar from the same release, with no checksum transcribed into the repository by hand
- [x] #3 The pinned version has exactly one source of truth: a version bump in the justfile changes what CI installs, with no second place to update
- [x] #4 A deliberately wrong checksum or a missing asset fails the workflow step loudly, proven by a negative control, rather than continuing to a gate that skips
- [x] #5 just setup and the hosted workflow agree on the same assets, so a clean local checkout and a clean CI run reach the same gate result
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 5: provision linux-amd64 envtest assets in the hosted Validate workflow from the justfile pin; verify the release sidecar checksum and fail-closed control; root lands provisioning with harness wiring in one commit and confirms admission ran in hosted logs.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 5 hosted run 34277495122 derived envtest 1.37.0 from the justfile, verified the Linux amd64 archive against its same-release sha512 sidecar, reported kube-apiserver Kubernetes v1.37.0 and etcd 3.7.0, exported KUBEBUILDER_ASSETS, and passed just check with 19 admission passes and one explicit-null leaf skipped. The corrupt-archive control failed loudly at checksum verification. Provisioning and harness wiring landed together in 510de1c193c00795d95448a705964707d1aad81f.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 5 provisioned checksum-verified envtest assets from the sole justfile pin and proved the admission gate runs in hosted CI at the integration SHA.
<!-- SECTION:FINAL_SUMMARY:END -->
