---
id: GCV-0097
title: Exclude linked-worktree Git metadata without weakening the public-release scan
status: In Progress
assignee: []
created_date: '2026-09-30 20:48'
updated_date: '2026-09-30 20:56'
labels:
  - needs-triage
dependencies: []
priority: high
type: bug
ordinal: 97000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Loop 22 isolated implementation cannot start: the public-release scan excludes Git directory descendants but scans a linked-worktree Git pointer file, whose machine-local administrative path is not publishable project content. Root gate passes in the normal checkout while the identical worktree gate fails before implementation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The public-release scan passes in a clean linked worktree with inherited identity checks enabled.
- [x] #2 Refused content in ordinary working-tree files remains rejected and reachable-history checks are unchanged.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Reproduce on a linked worktree, exclude only Git metadata at all six working-tree search sites, verify pass and negative leak control, run required local gate and CodeRabbit, push and collect exact-SHA hosted validation.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Loop 22 root prerequisite attempt GCV-0097-impl-1: reproduced clean linked-worktree scan failure before edit, exit1. After six metadata exclusions, clean scan exit0 and ordinary working-tree refused-path negative control exit1. History git-grep paths unchanged. Required just check exit0, final Validation passed. CodeRabbit completed with 0 findings and scripts/public-release-scan.sh reviewed. Hosted evidence pending exact completing push.
<!-- SECTION:NOTES:END -->
