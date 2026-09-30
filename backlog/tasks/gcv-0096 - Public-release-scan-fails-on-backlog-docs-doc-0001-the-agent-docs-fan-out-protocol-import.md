---
id: GCV-0096
title: >-
  Public-release scan fails on backlog/docs/doc-0001, the agent-docs fan-out
  protocol import
status: Done
assignee:
  - '@claude'
created_date: '2026-09-26 17:28'
updated_date: '2026-09-30 19:33'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 96000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The public-release scan (scripts/public-release-scan.sh, the first stage of just check / the hosted Validate public reference workflow) fails on backlog/docs/doc-0001 - Agent-fan-out-protocol-canonical.md, the verbatim import of m7kni/agent-docs's sources/fan-out-protocol.md. The doc carries a refused source API/domain identifier (a homelab host's metric-family names) and refused absolute local macOS home-directory paths (a tooling script under a user's local bin directory), both in the working tree and in reachable Git history, so a later commit removing them would not clear the history half of the scan. Do not quote the offending strings into this task, a commit message, or any tracked file while investigating - doing so adds a fresh, permanent occurrence to history; refer to commits by SHA instead. The hosted Validate public reference workflow has been red on every run since 2026-09-25 (confirmed via gh run list), all triggered by the automated docs: publish canonical agent documents sync commits. This is a different defect from GCV-0001 (the docs-hub repository reference false positive, already fixed in df56993) - no existing task tracks this one. Editing the doc copy in this repository is the wrong fix: AGENTS.md documents it as re-imported verbatim whenever upstream moves, so the next publish overwrites any local edit. The fix belongs either upstream, in m7kni/agent-docs's sources/fan-out-protocol.md, which must not carry host identifiers into public consumer repos, or in a deliberate narrowing of the scanner itself, in the style of GCV-0001's scan_fixed_allowing approach, if the identifiers are judged to be legitimate references rather than leaks.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The hosted Validate public reference workflow passes on a commit that resolves this, with the run ID recorded as evidence
- [x] #2 The fix is either an upstream change to m7kni/agent-docs sources/fan-out-protocol.md that stops carrying host identifiers into public consumer repos, or a deliberate, scoped scanner narrowing recorded next to the exclusion (not a blanket exclusion of the pattern or the file)
- [x] #3 The local copy of backlog/docs/doc-0001 is not hand-edited to route around the failure, since the next canonical-document publish would silently overwrite that edit
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-09-30: the canonical source was reworded in agent-docs 18059ca and republished to all consumers; this repository received it as 1a1b4eb, so the local doc copy was never hand-edited. The history occurrences are exempted in scripts/public-release-scan.sh by exact substrings only (the org-identifier phrase and one tool path, case-sensitive), in d1f3c0b; a negative control with another home-directory path and the bare identifier still failed the scan. Local just check passed; hosted Validate public reference run 36763862811 succeeded on f35c8a039ce79c7fa953cc4e2706e93da4430a42, the first green main run since the leak.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Reworded the leaked lines at the canonical source and republished, then exempted exactly those historical substrings in the publication scan with a recorded reason. Verified by a scan negative control, local just check, and hosted Validate run 36763862811 on f35c8a0.
<!-- SECTION:FINAL_SUMMARY:END -->
