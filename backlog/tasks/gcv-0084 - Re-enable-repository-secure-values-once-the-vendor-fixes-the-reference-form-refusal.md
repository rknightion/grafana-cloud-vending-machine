---
id: GCV-0084
title: >-
  Re-enable repository secure values once the vendor fixes the reference-form
  refusal
status: To Do
assignee: []
created_date: '2026-09-18 08:00'
labels:
  - needs-triage
  - vendor-defect
  - blocked-upstream
dependencies:
  - GCV-0081
type: feature
ordinal: 84000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0081 refuses spec.repository.secure.token, webhookSecret and commitSigningKey at admission because the only shape the API can express is the name-reference form the vendor refuses. The refusal is a holding action, not the intended end state: the fields stay defined so that re-enabling them is a rule removal rather than a schema change.

This task is the other half, and it is BLOCKED ON SOMEONE ELSE. The vendor acknowledged the defect on 2026-09-13 and said the fix needed coordination between teams; the inline create form was given as the interim workaround. No fix has been announced, and nothing in this repository will learn of one - there is no upstream issue to watch, because the owner decided not to file one, so the evidence trail is the vendor conversation and the live behaviour.

WHAT TO DO WHEN PICKING IT UP, in order, because the expensive mistake is trusting the first two steps.

Re-verify the refusal live before changing anything. The reference form may work now, or may work for Connection and not Repository, or may work only for some secure-value decrypters. Assume nothing from the recorded evidence, which will be months old.

Verify it for BOTH kinds. GCV-0081's refusal for Repository was inferred from Connection's observed 403, never observed directly, so the first live check is also the first direct evidence that Repository was ever affected. If it turns out Repository was never refused, the refusal shipped by GCV-0081 was wrong and that is worth knowing plainly rather than quietly reversing.

Remember the vendor's secret API ignores dryRun. Two objects posted with dryRun=All persisted as real resources the first time and had to be removed by hand. Every verification call mutates a live stack, so it goes on a stack nothing depends on and its residue is removed and recorded.

Then decide what happens to the connection API too. Its renderer emits the create form and still renders a standalone secure value that duplicates the credential inside the vendor, kept specifically so that restoring the reference form would be a one-line change. If the reference form works again, that is the change - and if it does not, that duplicate has no remaining justification. GCV-0082 carries the same decision from the other direction; settle it once, in whichever lands first.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The reference form is re-verified live for both the connection and repository kinds, on a stack nothing depends on, with residue removed and recorded
- [ ] #2 Whether the repository kind was ever actually refused is settled directly rather than by inference from the sibling kind
- [ ] #3 If the reference form works, the admission refusal is removed and the change is graded against the released-API rule
- [ ] #4 The connection renderer's duplicate secure value is resolved in the same change, either by restoring the reference form or by removing the duplicate
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
