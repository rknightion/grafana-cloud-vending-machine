---
id: GCV-0004
title: Bind signature verification to the installed package digest
status: To Do
assignee: []
created_date: '2026-08-20 16:10'
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
- [ ] #1 scripts/validate.sh fails when a verification job's digest argument differs from the package resource's spec.package
- [ ] #2 The function's verification job is a stable-named PreSync hook, matching the provider's
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
