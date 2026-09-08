---
id: GCV-0034
title: 'Admit every XRD against a real API server so CEL rules are proven, not parsed'
status: Done
assignee: []
created_date: '2026-09-08 17:02'
updated_date: '2026-09-08 21:01'
labels: []
dependencies: []
ordinal: 34000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every wave-3 lane closed with the same unproven line: the XRD CEL validation rules were parsed and rendered locally, never admitted by a Kubernetes API server. Thirteen XRDs now carry structural schemas, x-kubernetes-validations rules, immutability rules and preserve-unknown-fields subtrees whose real behaviour under apiserver CEL, pruning and ratcheting has never been observed. A locally parsed rule that the apiserver rejects at install time, or silently prunes before evaluating, is indistinguishable from a working one under the current gate. An ephemeral local control plane is not a live Grafana Cloud stack and does not breach the no-live-environment constraint.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every XRD in platform/apis/ is installed into an ephemeral local Kubernetes API server and accepted without error
- [x] #2 Each XRD's structural schema is confirmed accepted by the apiserver, including every x-kubernetes-validations rule, so a CEL expression that fails apiserver compilation fails the harness
- [x] #3 A valid example request for each API is admitted through the real apiserver, not merely parsed
- [x] #4 The harness is reproducible from a clean checkout with one just recipe and pins its control-plane version explicitly
- [x] #5 The recipe is placed in check or ci according to the frozen task-surface rule, with a comment naming which heavy dependency it needs if it lands in ci
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 4: implement the commissioned lane after the pushed root harness pre-pass; preserve frozen schemas and ownership; return acceptance evidence and required negative controls; root integrates, reviews, validates locally and at the exact hosted SHA, then reconciles status.

Wave 5: resume the preserved envtest harness after the root lands the one-line collectionRefs repair; prove 14/14 derived CRDs, every XRD-backed catalog example, corrupt-CEL and reverted-collectionRefs controls; root wires it into check and verifies local and hosted execution.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 4 admission attempt is blocked by an existing request-schema defect. The source census is 13 YAML paths containing 14 XRD documents, including both access APIs. A real local Kubernetes 1.37.0 API server, using the pinned Crossplane ForCompositeResource conversion without altering nested schemas, installed 13/14 CRDs. The remaining agent-observability-v1beta1.yaml CRD was rejected at spec.guards.ruleActions[].collectionRefs[]: x-kubernetes-map-type must be atomic when the parent list has x-kubernetes-list-type=set. Corrupting stack-v1beta1.yaml CEL to self.metadata.name == was rejected by the API server with its compilation error; restoring the scratch schema was accepted. A narrow diagnostic admitted 13 catalog examples and skipped 2 objects (the blocked API example and a Composition input with no XRD).

No existing XRD may change under this wave's grant, so full installation and the mandatory admission gate are not delivered. Main retains the false pre-pass readiness seam and one skipped placeholder; a green existing gate is not admission proof. The owned implementation is preserved locally under codex/wave4/lane-A-artifact/ with A-admission-verbatim.log and A-catalog-diagnostic.log. Root wiring would promote existing crossplane-runtime and apimachinery imports and add the recorded test-only transitive checksum entries.

Resume: explicitly authorize the structural-schema repair; rerun the preserved faithful all-XRD and catalog tests unchanged; require all 14 CRDs and every XRD-backed example; then flip readiness, integrate lane B, provision checksum-verified envtest assets locally and in hosted validation, and wire the admission recipe into just check. No acceptance or Definition of Done item is claimed complete.

## Resume authority granted - 2026-09-08 (wave 4 review)

The repository owner authorized the structural-schema repair at the wave 4 review. The repair is bounded to exactly one line in platform/apis/agent-observability-v1beta1.yaml: add x-kubernetes-map-type: atomic to the items object of spec.guards.ruleActions[].collectionRefs. It is non-breaking, preserves apiserver-enforced set dedupe, and leaves the request shape and the published migration guide untouched. Collapsing the item to a string set and dropping the set constraint were both offered and both rejected.

Reviewer finding that raises this above a harness blocker: the derived CRD is invalid, so this XRD cannot install into ANY Kubernetes cluster today, not only the test harness. The published reference currently ships an uninstallable API. A reviewer sweep of all 13 XRD files found 35 list-type declarations (16 set, 18 map, 1 atomic) and exactly this one violation; GCV-0040 now covers the class statically.

No other schema change is authorized by this grant.

## Hosted-CI prerequisite found at the wave 4 review - 2026-09-08

Wiring the harness into scripts/validate.sh makes the hosted Validate public reference workflow fail, because that workflow installs only ripgrep, Go and just, and KUBEBUILDER_ASSETS is asserted solely in just setup which CI never calls. The harness is correctly forbidden from skipping when its binaries are absent, so this surfaces as a red hosted run rather than a silent pass. This task's DoD item 2 therefore cannot be met until GCV-0041 lands. GCV-0041 is now a recorded dependency and both are commissioned in the same wave.

Wave 5 delivered the envtest harness in just check. Thirteen XRD source files derived fourteen CRDs; the API server installed 14/14 and admitted 25 XRD-backed catalog objects, with the sole excluded GrafanaVendingConfig object recorded as Composition input. Corrupt-CEL and missing-atomic-map controls failed through the API server and passed after restore. Local and hosted gates each recorded 19 admission passes and one explicit-null leaf skipped. Hosted Validate run 34277495122 succeeded at 510de1c193c00795d95448a705964707d1aad81f.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Parked after real API-server evidence exposed a pre-existing schema defect outside the wave mutation boundary. The concrete repair and unchanged-test resume requirements are recorded above.

Wave 5 delivered a pinned real-API-server admission harness in just check. Verified 14/14 derived CRDs, all 25 XRD-backed catalog objects, both API-server negative controls, and the exact-SHA hosted gate.
<!-- SECTION:FINAL_SUMMARY:END -->
