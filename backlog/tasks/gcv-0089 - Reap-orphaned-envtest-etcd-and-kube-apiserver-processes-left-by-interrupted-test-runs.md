---
id: GCV-0089
title: >-
  Reap orphaned envtest etcd and kube-apiserver processes left by interrupted
  test runs
status: To Do
assignee: []
created_date: '2026-09-19 18:05'
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
