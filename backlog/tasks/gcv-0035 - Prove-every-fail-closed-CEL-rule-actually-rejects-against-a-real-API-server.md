---
id: GCV-0035
title: 'Prove every fail-closed CEL rule actually rejects, against a real API server'
status: Parked
assignee: []
created_date: '2026-09-08 17:02'
updated_date: '2026-09-08 22:25'
labels: []
dependencies:
  - GCV-0034
ordinal: 35000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The vending machine's safety story rests on rules that are asserted to reject: SCIM presence, creation-time-only retention and expiry, immutable ladder identity and ordering, append-only extension records, explicitly empty subnet restrictions, omitted k6 zone intent. Each is currently proven only by a Go unit test over the renderer or by reading the YAML. Admission is the layer that actually enforces them, and a rule that compiles is not a rule that rejects: a mistyped field path, a pruned subtree or a rule attached at the wrong level all admit the request the platform intended to refuse. The negative case is the one that matters here, because every one of these rules exists to be the last line before a credential, a deletion or a billing surface.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each fail-closed rule has a table-driven case that submits the forbidden request to the real apiserver and asserts admission is refused
- [ ] #2 Each rejection case asserts on the rule's own message, so a request refused for an unrelated reason cannot pass as proof
- [x] #3 The paired allowed case is admitted for every rule, so no case passes by rejecting everything
- [x] #4 Immutability rules are proven by admitting a create then submitting the forbidden update, not by a create alone
- [x] #5 A deliberately weakened rule is shown to fail the harness, recorded as a negative control in the task summary
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 4: implement the commissioned lane after the pushed root harness pre-pass; preserve frozen schemas and ownership; return acceptance evidence and required negative controls; root integrates, reviews, validates locally and at the exact hosted SHA, then reconciles status.

Wave 5: resume the preserved 16-case admission table; remove only nullable: true from spec.scim; accept the repair only if the API server errors rather than prunes explicit null; re-prove allowed pairs, immutable updates, tight messages, and weaken/restore control.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 4 real-apiserver diagnostic ran 16 leaf tests: 15 passed, 1 failed, 0 skipped. These comprise 15 rejection variants plus one weaken/restore control. Fourteen forbidden variants rejected with their own rule messages; every paired allowed case admitted, and immutability was exercised as admitted create followed by update. Explicit spec.scim: null was admitted. The nullable field and !has(self.scim) rule cannot reject that input because CEL treats null-valued fields as absent. The renderer still has a defensive presence check; this finding is specifically an admission gap.

Negative control, verbatim: baseline output: SCIM is out of scope; use external-group mapping. Weakened scratch rule output: admitted. Restored rule output: SCIM is out of scope; use external-group mapping. Full API-server errors and all cases are preserved in codex/wave4/B-admission-rules-real.log; the owned test artifact is codex/wave4/lane-B-artifact.go. The unrelated subnet restriction belongs to GrafanaVendingConfig Composition input, not an XRD: both focused configured-profile and explicit-empty renderer tests passed. That earlier command also contained one skipped pre-pass placeholder, separately from the later zero-skip real suite.

Main has not received the implementation and still has the false readiness seam. Resume: explicitly authorize a presence/schema design that preserves omission and rejects explicit null; rerun the unchanged null case plus all rejection/allowed/update pairs and the weaken/restore control, then integrate with GCV-0034. Removing nullable alone is not a verified fix because pruning must be tested. No completing hosted validation or delivered admission gate is claimed.

## Resume authority granted - 2026-09-08 (wave 4 review)

The repository owner authorized removing nullable: true from spec.scim in platform/apis/stack-v1beta1.yaml, so that an explicit spec.scim: null fails structural type validation instead of slipping past !has(self.scim). Accepting explicit null as equivalent to omission, and attaching a property-level CEL rule to the nullable field, were both offered and both rejected.

The grant is conditional and the condition is the acceptance check, not a caveat: the repair is only accepted once a test proves the API server ERRORS on explicit null rather than PRUNING it. A pruned field is admitted, which is the same failure under a different mechanism. Prove the behaviour before claiming the fix. Assert on the resulting message, whatever it is - a structural type error is a different message from the SCIM rule message, so AC #2's own-message requirement is satisfied by the type error for this one case and by the rule message for the other fourteen.

No other schema change is authorized by this grant.

Wave 5 exercised the owner-authorized nullable removal against Kubernetes 1.37. The API server pruned and admitted explicit spec.scim: null instead of returning a type error, twice with the same signature, so the candidate schema line was rejected and the lane parked under the conditional grant. Fourteen refusal variants, every paired allowed case, every admitted-create then refused-update sequence, and the weaken/admit/restore negative control passed and were integrated. The explicit-null leaf remains visibly skipped with the park reason. Resume only after owner authorization for another schema design; accepting null as omission and property-level CEL remain rejected.

## Reviewer correction to the recorded mechanism - 2026-09-08

The wave 5 park is correct and stands. Its recorded reason was WRONG and is corrected here, because a wrong mechanism sends the next attempt down a dead end.

WRONG: "Kubernetes 1.37 prunes and admits explicit spec.scim: null."
TRUE: that describes the REJECTED candidate schema, with nullable removed. Under the SHIPPED nullable: true schema the API server PERSISTS the null; it does not prune it. Proven against a real API server at the pinned envtest version:

  PERSISTED explicit-null: spec has 'scim' key = true, value = null
  PERSISTED omitted:       spec has 'scim' key = false

So the admission gap is not pruning. It is that CEL has() does not see a null-valued field, and !has(self.scim) therefore returns true.

Two further designs were probed against the same API server and both are dead:

- A property-level x-kubernetes-validations rule on the nullable scim field rejects spec.scim: {} but still ADMITS spec.scim: null, so the rule does not evaluate for an explicit null. This was one of the two alternatives rejected by the owner at the wave 4 review; it would not have worked in any case.
- !('scim' in self) does not compile. The API server refuses the XRD with "found no matching overload for '@in' applied to '(string, selfType...)'", so @in has no overload for an object-typed self in CRD CEL.

The safety consequence is materially better than the report implied. Because the null is persisted with its key present, renderStack's Go map lookup at platform/function/fn.go:453 DOES see it and refuses the request, and platform/function/scim_test.go already covers the nil value explicitly. The platform therefore fails closed; the only real gap is that the refusal happens at reconcile rather than at kubectl apply.

The skip reason in platform/function/admission_rules_test.go was corrected to match, in commit baa4a27.

Resume boundary, restated: no CRD schema shape found so far can refuse an explicit spec.scim: null at admission. Anything that would requires either accepting the reconcile-time refusal as sufficient, or a validating admission webhook that reads the raw request before CEL. Both are owner decisions.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Parked after real API-server evidence exposed a pre-existing schema defect outside the wave mutation boundary. The concrete repair and unchanged-test resume requirements are recorded above.

Wave 5 parked because removing nullable did not reject explicit spec.scim: null: Kubernetes 1.37 pruned and admitted it. Fourteen other refusal variants and the negative control are delivered; AC 1 and 2 remain unmet for the explicit-null case.
<!-- SECTION:FINAL_SUMMARY:END -->
