---
id: GCV-0039
title: >-
  Resolve the open SCIM user-lifecycle question and correct the recorded plan
  floor
status: Parked
assignee: []
created_date: '2026-09-08 17:03'
updated_date: '2026-09-08 18:39'
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
- [x] #1 The plan-floor correction is recorded against GCV-0028 as a superseding note, with the older statement left intact as history rather than edited away
- [x] #2 The user-lifecycle question is answered from an authoritative dated source, or recorded as still unresolved with the exact evidence that would settle it
- [x] #3 Where the answer is unresolved the task is Parked with a concrete resume boundary, not closed as Done
- [x] #4 No live tenant, Grafana Cloud API or identity provider is contacted, and no source-environment identifier enters the tracker
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 4: implement the commissioned lane after the pushed root harness pre-pass; preserve frozen schemas and ownership; return acceptance evidence and required negative controls; root integrates, reviews, validates locally and at the exact hosted SHA, then reconciles status.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Superseding correction - 2026-09-08

This note supersedes only the historical plan-floor statement. The repository owner has decided that SCIM is available on all Grafana Cloud plans; there is no Pro-and-above or Advanced-only entitlement floor. Earlier plan-floor wording remains as historical context and must not guide future admission decisions.

The lifecycle question remains unresolved: current Grafana documentation describes `user_sync_enabled` only while it is enabled, when SCIM requests can create, update, and deactivate users. It also says a SCIM-provisioned user cannot be deleted and can be deactivated through the identity provider. It does not state whether disabling `user_sync_enabled` removes, deactivates, suspends, or leaves already-provisioned users unchanged. Do not infer any of those outcomes.

This is settled only by either an authoritative, dated Grafana Cloud statement explicitly describing the post-disable state of already-provisioned users, or a controlled disposable Grafana Cloud test that records the same SCIM user before and after disabling user sync, without an identity-provider membership or provisioning change, and verifies the user record, active state, and authentication result.

Evidence collected in wave 4: ## Sources
- https://grafana.com/docs/grafana/latest/setup-grafana/configure-access/configure-scim-provisioning/ - strong primary for enabled behavior, silent on disabled transition.
- https://grafana.com/docs/grafana/latest/setup-grafana/configure-access/configure-scim-provisioning/manage-users-teams/ - strong primary for IdP-driven deactivation, insufficient for Grafana-side switch.
- Both source paths on release-13.2.2 last committed 2026-02-20T10:18:38Z, commit4bb6b7f (link fix only). Source date does not strengthen missing claim.
- Context7 found no matching Grafana Cloud disabled-sync lifecycle docs; negative discovery only.

No live tenant, Grafana Cloud API, identity provider or secret store was contacted. This unresolved lifecycle is the explicitly permitted Parked outcome. The plan-floor note has been appended to GCV-0028 without rewriting its older statements.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Parked correctly: all-plan availability correction recorded; disabled-user-sync lifecycle remains unresolved. Resume only with the exact dated vendor statement or separately authorized controlled disposable evidence specified in the notes.

Completing checkpoint SHA 5c482ffcf2100714cfe1c3751733d805a501489a; hosted Validate 34263860085 success. Implementation SHA 9c559d105c5cd7761db3c9ca290150d35c936175; hosted Validate 34263352393 success. Local just check passed with 85.4% coverage. Main still has zero real admission cases and one skipped placeholder; these tasks do not claim admission completion. The all-plan correction is published and the unresolved lifecycle remains Parked as required.
<!-- SECTION:FINAL_SUMMARY:END -->
