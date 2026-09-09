---
id: GCV-0064
title: >-
  Review the wave 9 corrections that landed after the CodeRabbit two-pass
  ceiling
status: Done
assignee:
  - '@claude'
created_date: '2026-09-09 21:28'
updated_date: '2026-09-09 22:50'
labels: []
dependencies: []
ordinal: 64000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Wave 9 spent both CodeRabbit passes before its two most load-bearing corrections existed. The R16 terminal-dispatch fixes (argument-traversal preservation, parser-resolved switch receiver matching, first-result certification) and the R17 provider identity delimiter correction both landed after the second service snapshot. The security lane reviewed them and the full gate passed, so they are not unreviewed - but the review that catches a different defect class than a security read did not see those bytes. GCV-0059 was opened for the same shape after wave 6 and found real findings, which is why this is worth the pass rather than assuming the gate covered it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A CodeRabbit review has run against the wave 9 diff range that contains the R16 and R17 corrections
- [x] #2 Every critical and major finding is fixed
- [x] #3 Each finding below major is either fixed or left with a recorded reason naming why it is not impactful in context
- [x] #4 The review command, its base, the finding counts by severity and the disposition of each finding are recorded in the task notes
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Confirm the exact commit range holding the R16 and R17 corrections and that CodeRabbit has not already reviewed those bytes.
2. Run the CodeRabbit review against that range, scoped to the composition function directory the corrections live in.
3. Read every finding regardless of severity band; fix critical and major without judgement, and decide each lower finding on its impact in this code.
4. Re-run the full gate after any fix.
5. Record the command, base, severity counts and per-finding disposition in the notes.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Review run: `coderabbit review --agent --base 90094fdc1bfe32666e37d816ecc0bc993a2323e7 --dir platform/function`, against local HEAD 81051ca77548449550ffa40add213d172609ef55. The base is the last commit before the wave 9 implementation commit, so the range covers every byte the wave produced, including both corrections this task exists for.

The run emitted a `complete` event, so it is a real review rather than an aborted one. Findings: 0. Severity counts are therefore 0 critical, 0 major, 0 minor, 0 trivial, 0 info, and no finding needed a disposition.

Reviewed files, 10: admission_harness_test.go, family_docs_adversarial_test.go, family_docs_test.go, fn.go, install.yaml, stackconsumer.go, stackconsumer_admission_test.go, stackconsumer_adversarial_test.go, stackconsumer_test.go, testdata/provider-crds.json.

Both target corrections are inside that set. The R16 terminal-dispatch repairs live in the family traversal and its controls (family_docs_test.go, family_docs_adversarial_test.go) and in fn.go; the R17 provider identity delimiter comparator is stackconsumer.go, which compares the observed external name against `region + ":" + id`. So the bytes the two-pass ceiling never saw have now been reviewed.

Independent confirmation of R17 while verifying this task: the pinned upstream terraform-provider-grafana v4.45.1 defines `ResourceIDSeparator = ":"` in internal/common/resource_id.go, which is the colon form the corrected comparator uses. The correction was right and the pre-correction slash form would have been wrong.

R17 provenance, to the standard the wave operating model requires of a pinned-source citation:

- upstream: grafana/terraform-provider-grafana, tag v4.45.1, commit 7f3311691b0e124c55347f7204501fba77f61453
- file: internal/common/resource_id.go, SHA-256 e6e9ac299f849b65d5785a0c3bf37706292ee5e7e8cd9f517361368666193a95
- line 13 declares `ResourceIDSeparator = ":"`; line 107 joins the ID parts with it.

That is the separator the corrected comparator in platform/function/stackconsumer.go uses when it
builds `region + ":" + id`. The pre-correction slash form would never have matched a real
provider-assigned AccessPolicy external name, and no synthetic fixture could have exposed that,
because the fixture supplied the same wrong form the comparator expected.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Ran `coderabbit review --agent --base 90094fdc1bfe32666e37d816ecc0bc993a2323e7 --dir platform/function`, covering the whole wave 9 change including the R16 terminal-dispatch repairs and the R17 provider identity delimiter correction, both of which landed after the wave spent its second and final CodeRabbit pass. The run emitted a complete event over 10 files and returned zero findings across every severity band, so nothing needed fixing or a recorded disposition.

Independently confirmed R17 against the pinned upstream while verifying: grafana/terraform-provider-grafana v4.45.1, commit 7f3311691b0e124c55347f7204501fba77f61453, internal/common/resource_id.go (SHA-256 e6e9ac299f849b65d5785a0c3bf37706292ee5e7e8cd9f517361368666193a95) declares `ResourceIDSeparator = ":"`. That is the separator the corrected comparator uses. The correction was right, and no synthetic fixture could have exposed the original error because the fixture supplied the same wrong form the comparator expected.

Completing SHA 70a896c7a63b222bcbbbfad041d605b8b2643abe; hosted Validate public reference run 34413634975 success; local `just check` green on the same SHA at 84.9% coverage.
<!-- SECTION:FINAL_SUMMARY:END -->
