---
id: GCV-0004
title: Bind signature verification to the installed package digest
status: Done
assignee: []
created_date: '2026-08-20 16:10'
updated_date: '2026-08-20 16:20'
labels:
  - supply-chain
dependencies: []
type: bug
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Each package's immutable digest is written twice: once in spec.package on the Function or Provider, and once in the argv of the cosign job that verifies it. Nothing linked the two, and the gate did not look at either, so editing the package pin and forgetting the job left a verification that passed green against the previous digest. That is the one failure mode the verification exists to prevent.

The function's job compounded it by carrying a truncated digest in its own metadata.name, so a bump produced a differently-named job and left no trace that the old one had verified something else.

Fix: assert in the gate that the digest in argv equals spec.package, for both packages, and give the function's job a stable name as a PreSync hook with BeforeHookCreation and HookSucceeded, exactly as the provider's job is configured, so it re-verifies on every sync rather than once per digest.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 scripts/validate.sh fails when a verification job's digest argument differs from the package resource's spec.package
- [x] #2 The function's verification job is a stable-named PreSync hook, matching the provider's
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
scripts/validate.sh now loads both package files, takes the first argv entry containing @sha256: from the verification Job and compares it with spec.package on the Function or Provider, aborting with both values when they differ. Proven by seeding a mismatched digest: the gate stopped with 'platform/function/install.yaml: verifies ...673028c1... but installs ...0000...' and did not print its success line.

The function's Job is now named verify-function-grafana-vending with argocd.argoproj.io/hook: PreSync and hook-delete-policy: BeforeHookCreation,HookSucceeded at wave 0, matching the provider's. The digest is gone from the Job name, so there is one fewer place to forget, and verification runs on every sync rather than once per digest.

Confirmed live: the hook ran during the sync that installed the new function digest, and the Job is absent afterwards because HookSucceeded deleted it, while crossplane-platform reports Synced/Healthy -- a failed PreSync hook would have blocked the sync instead.

Completing SHA 9a2b73d, hosted validation run 32390912362.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
The gate asserts that what a verification job verifies is what the package resource installs, proven against a seeded mismatch, and the function's job now matches the provider's as a stable-named PreSync hook that re-verifies on every sync.
<!-- SECTION:FINAL_SUMMARY:END -->
