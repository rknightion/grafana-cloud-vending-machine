---
id: GCV-0089
title: >-
  Reap orphaned envtest etcd and kube-apiserver processes left by interrupted
  test runs
status: In Progress
assignee:
  - '@codex'
created_date: '2026-09-19 18:05'
updated_date: '2026-09-24 08:39'
labels: []
dependencies: []
ordinal: 89000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
An interrupted envtest run leaves its etcd and kube-apiserver alive indefinitely. Two orphaned pairs were found on a developer Mac, resident 31h and 23h after their runs had gone, holding roughly 250-280MB RSS per pair plus a 123MB temp data directory each under the system temp dir as k8s_test_framework_*. Two further k8s_test_framework_* directories were present with no owning process at all, so the leak accumulates across runs.

The processes were killed manually. envtest tears these down when the test binary exits normally; it does not when the run is interrupted, panics, or the harness is killed.

Not a disk-throughput problem: envtest starts etcd with --unsafe-no-fsync=true and preallocates a 64MB WAL that never rotates, and the host disk measured 0.03 MB/s at idle with all four processes running. The cost is resident memory, held PIDs and loopback ports, and unbounded temp-directory growth.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A check reports any envtest etcd or kube-apiserver process whose elapsed time exceeds a plausible ceiling for a test run
- [ ] #2 A reap path kills orphaned envtest processes and removes their k8s_test_framework_* temp directories, and refuses to touch a pair belonging to a live test run
- [ ] #3 k8s_test_framework_* directories with no owning process are reported and reclaimed separately, since those leak disk with no process to find
- [ ] #4 The reap is reachable from the justfile as a named recipe in the correct group, and does not become a new scripts/*.sh task runner
- [ ] #5 Verified by interrupting an envtest run on purpose, then confirming the check fires and the reap reclaims both the processes and the directory
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 16 lane B adds a justfile-only check/reap interface, proves it refuses live runs, and verifies interrupted envtest cleanup under repository-pinned assets.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 17 attempt 2 refactored the candidate into scripts/envtest-processes.sh with justfile entry recipes and passed syntax/format/dump checks, but runtime proof was blocked twice by asset-scoped envtest processes outside the lane's owned TMPDIR and outside the empty campaign start witness. The frozen shared-resource rule forbade terminating them. No process or directory was removed; deliberate interruption, live-parent refusal, orphan pair reap, ownerless-directory reclaim and symlink containment remain unproven. Candidate blobs remain unlanded in the working tree: justfile 494524c74884bb4fa804461e26115ebdbae230cc and script 3f5a51faadb8d8f6c4b4a4166836ed6b820f0bb3.

Loop 18 attempt 3 runs reaper proofs with lane-private copied envtest assets and lane-private TMPDIR. The discriminating check must list only lane-owned PIDs.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 16 lane B was interrupted by the mandatory repeated-review stop while its justfile-only check/reap implementation and verification were still in progress. No integrated gate, commit or hosted validation exists, and no acceptance criterion is recorded complete. Resume by auditing the working-tree recipe, confirming no repository-pinned envtest process or k8s_test_framework directory was left behind, replaying live-parent refusal plus deliberate interruption/reap evidence under an empty kubeconfig, and then running the integrated gate.

Wave 17 left the envtest reap candidate unlanded because the required deliberate runtime proof could not start under the shared-process safety rule. Static checks passed; no process was terminated and AC1-AC3/AC5 plus both DoD items remain open.
<!-- SECTION:FINAL_SUMMARY:END -->
