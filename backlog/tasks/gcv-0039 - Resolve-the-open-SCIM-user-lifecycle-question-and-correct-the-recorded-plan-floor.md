---
id: GCV-0039
title: >-
  Resolve the open SCIM user-lifecycle question and correct the recorded plan
  floor
status: To Do
assignee: []
created_date: '2026-09-08 17:03'
labels: []
dependencies: []
ordinal: 39000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0028 decided SCIM stays out of scope and closed with two explicitly unresolved product questions. One is now settled by the repository owner on 2026-09-08: SCIM is available on all Grafana Cloud plans, so there is no Pro-and-above or Advanced-only entitlement floor. Published documentation may still state an older floor and must not be treated as authoritative against this. The second question is open and is the one that actually gates any future adoption: when user sync is disabled, what happens to already-provisioned users, are they removed, disabled, suspended, or left unchanged and frozen. The pinned provider schema carries no lifecycle guarantee either way, so this cannot be answered from the code. It matters because the answer decides whether disabling sync is a reversible configuration change or a destructive one, and that changes whether SCIM could ever be vended by a machine that must fail closed. Answering it does not reopen the out-of-scope decision.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The plan-floor correction is recorded against GCV-0028 as a superseding note, with the older statement left intact as history rather than edited away
- [ ] #2 The user-lifecycle question is answered from an authoritative dated source, or recorded as still unresolved with the exact evidence that would settle it
- [ ] #3 Where the answer is unresolved the task is Parked with a concrete resume boundary, not closed as Done
- [ ] #4 No live tenant, Grafana Cloud API or identity provider is contacted, and no source-environment identifier enters the tracker
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
