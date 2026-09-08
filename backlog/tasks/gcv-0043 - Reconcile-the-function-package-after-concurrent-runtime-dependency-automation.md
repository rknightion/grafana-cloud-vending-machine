---
id: GCV-0043
title: Reconcile the function package after concurrent runtime dependency automation
status: Done
assignee: []
created_date: '2026-09-08 21:34'
updated_date: '2026-09-08 22:25'
labels:
  - needs-triage
dependencies: []
type: bug
ordinal: 43000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A concurrent runtime dependency update landed while wave 5 was in flight. The publisher's runtime-change detector ran from the function subdirectory and incorrectly reported that the change did not affect runtime inputs, so no package was published and the immutable package references were not moved. Reconcile the retained source and package state before release.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Decide whether the concurrent dependency bump remains the intended function source.
- [x] #2 If retained, publish and sign from the exact source SHA and verify workflow identity.
- [x] #3 Move all function package digest references together to the verified immutable digest.
- [x] #4 Prove just check and exact-SHA hosted validation.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closeout update: concurrent security dependency automation merged as source SHA 629c39c2b42e6294df0bdc3c59442846940779f0. The corrected runtime-change detector selected that source; publisher run 34282605376 completed its test, vet, amd64 build, arm64 build, multi-platform push, and signature steps for immutable tag v0.0.0-629c39c2b42e. The checked-in function package references still point to the previous digest. Resume by resolving the new tag to its immutable digest, verifying its signature and source/run identity, then moving both package references together. Do not republish unless that verification fails.

## Closed on main during the wave 5 review cycle - 2026-09-08

AC1: the concurrent bump is retained. It is google.golang.org/grpc v1.83.2, a security update merged as source SHA 629c39c2b42e6294df0bdc3c59442846940779f0. Nothing about it makes it unintended function source.

AC2: no republish was needed and none was performed. Publisher run 34282605376 had already published and signed immutable tag v0.0.0-629c39c2b42e from that exact SHA; its Test function, Build amd64 package, Build arm64 package, and Publish and sign package jobs all completed success. The signature was verified independently before the pin moved:

  Verification for .../function-grafana-vending@sha256:2ae81f6ec64b1caa6f53f30d51f77fc3cc282b4e4a6d92abf0a7384f67636c48
    - The cosign claims were validated
    - Existence of the claims in the transparency log was verified offline
    - The code-signing certificate was verified using trusted certificate authority certificates

The workflow identity was bound both ways. A deliberately wrong --certificate-github-workflow-sha was rejected with "expected GithubWorkflowSHA to be <wrong>, got 629c39c2b42e6294df0bdc3c59442846940779f0", and the correct SHA plus --certificate-github-workflow-repository=rknightion/grafana-cloud-vending-machine then verified. The certificate therefore pins the digest to this repository and to that exact source commit; the value was read out of the certificate rather than transcribed from a log.

AC3: both digest references in platform/function/install.yaml, the cosign verifier argument and the Function package field, moved together in a single commit 8b602cc. No other pin was touched; the provider pin remains v2.14.0 at sha256:3f078cf9f0fa4affbf65a5d9e288d3a33fced0b8e79354589763f1e7bd7c7048.

AC4: local just check passed with the pinned envtest assets, ending "coverage: 85.7% of statements" and "Validation passed.", with 19 admission tests passing and the single explicit-null leaf skipped. Hosted Validate run 34285644483 completed success at the exact completing SHA baa4a27f360a3c254aa516eeef9f9fbb789dd1cd.

The corrected runtime-change detector was confirmed live in the same push: a test-only Go change reported "Runtime inputs changed: false" in publisher run 34285644471 and the build and publish jobs correctly did not run.
<!-- SECTION:NOTES:END -->
