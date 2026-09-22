---
id: GCV-0090
title: >-
  Renovate cannot update the pinned-versions table, so every version-bearing
  component bump red-lights the gate
status: Done
assignee:
  - '@codex'
created_date: '2026-09-22 15:56'
updated_date: '2026-09-22 19:18'
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
- [x] #1 The root cause is confirmed by reproducing a component version bump in the manifest alone and observing the exact validator abort, not inferred from PR 46's log
- [x] #2 The chosen repair is recorded with the alternative considered, including what the pinned-versions table still publicly guarantees after the change
- [x] #3 Every pinned-versions row is audited for a version-bearing locator, so the fix covers the class rather than the cosign row
- [x] #4 A manifest-only version bump passes the gate with no markdown edit, proven by a commit that changes only the manifest
- [x] #5 A genuinely drifted table still fails the gate by path and component, proven by a negative control
- [x] #6 PR 46 reaches a green hosted Validate run without a hand-written documentation edit, or the reason it still cannot is recorded
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 16 lane A owns scripts/validate.sh, renovate.json, and docs/installation.md; reproduce drift, choose and record repair, add negative controls, then hand back to root for independent security review and integrated gate.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 17 independent phase-1 security review replayed all 56 cases: 47 negatives rejected and nine positives passed against validator blob e67f11cdd822678d677900f6329a958f3bc3ef41. The initial base reproduction emitted the exact documented Cosign abort; version-free locators and component/path/per-source negatives cover the class. Unpushed manifest-only proof commit 69740c5 changed only platform/function/install.yaml and platform/provider/provider-grafana.yaml from v3.1.2 to v3.1.3 and passed KUBECONFIG=/dev/null just check with Validation passed and 85.3% coverage. Pull request 46 remained open/red at Renovate head 4dadb6cc5f2fd4366c16653901fcb56e85e28381 and requires Renovate's own rebase; this run made no PR write. Source commit ac465f9a842089d8588259ee72ff8125ba236405 is contained in completing main SHA 5f672c09a9fc162fcfbe28ad42d342475cb2a798; hosted Validate run 35771159008 succeeded.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 16 selected the version-free locator design. Lane A demonstrated a manifest-only Cosign bump passing and a wrong-component version failing after correction. The source remains uncommitted because the mandatory campaign stop occurred before lane E completed the required exhaustive replay of every pre-existing validator negative control. Resume by replaying and recording all historical negative controls against the corrected validator, then run the integrated and hosted gates. Pull request 46 still requires Renovates own scheduled rebase after a landed fix; this run made no pull-request write.

Made pinned-version validation derive explicit versions from each named source while permitting version-free locators, retained fail-closed component/path/per-source drift controls, and proved a real manifest-only Cosign bump in unpushed commit 69740c5. Completed by source commit ac465f9a842089d8588259ee72ff8125ba236405 within main SHA 5f672c09a9fc162fcfbe28ad42d342475cb2a798; hosted Validate run 35771159008 passed. PR 46 still needs Renovate's own rebase.
<!-- SECTION:FINAL_SUMMARY:END -->
