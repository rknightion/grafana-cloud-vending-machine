---
id: GCV-0006
title: Give the API a reviewable decommission path
status: To Do
assignee: []
created_date: '2026-08-20 16:10'
labels:
  - api
  - design
dependencies: []
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
managementPolicies (Create, Observe, Update, LateInitialize) and the Stack's deleteProtection: true are package-level constants in the function with no API surface. Deleting a stack for real therefore cannot be expressed as a request at all.

What it takes today, proven on a live control plane retiring two stacks: annotate each request crossplane.io/paused=true so the composition cannot revert the next step; patch every org-level managed resource to managementPolicies ['*']; patch the Stack's forProvider.deleteProtection to false; wait for status.atProvider to confirm protection is off externally rather than only in spec; remove the request from Git; then remove the pause annotation so the finalizer clears.

Every one of those steps is a kubectl call against a composed resource. None appears in a diff, none is reviewable, and the operator has to work out for themselves which resources hold external state that outlives the stack. Get that last part wrong and the visible outcome is a clean deletion while an organization-level access policy and its live telemetry-publisher token stay behind with nothing owning them. Stack-local resources are safe to orphan because deleting the stack takes them; organization-level ones are not, and nothing in the repository says so.

The principle that there is no one-command destructive path is worth keeping. A lifecycle field does keep it: arming deletion is one reviewable change and removing the request is a second, and both are visible to a reviewer, which the current sequence is not.

Design questions to settle first: whether the field belongs on the request or is inferred from the profile; whether deleteProtection and management policies move together or separately; whether an armed-but-not-deleted request should surface a condition; and what happens to generated secret-store documents, which today are retained by deletionPolicy and were deleted by hand.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A platform-owned lifecycle field on GrafanaCloudStackRequest selects between retaining and deleting external resources, defaulting to retain
- [ ] #2 Which profiles may select deletion is a platform decision, not a consumer one
- [ ] #3 Arming deletion and removing the request stay separate changes, so there is still no one-command destructive path
- [ ] #4 The decommission runbook is rewritten around the field, and the manual patching sequence is deleted
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
