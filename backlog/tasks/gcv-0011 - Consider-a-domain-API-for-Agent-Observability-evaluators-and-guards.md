---
id: GCV-0011
title: Consider a domain API for Agent Observability evaluators and guards
status: To Do
assignee: []
created_date: '2026-08-21 12:14'
updated_date: '2026-08-27 09:27'
labels: []
dependencies:
  - GCV-0010
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Once the provider pin carries the agento11y family, five kinds become available: Collection, Evaluator, EvaluationRule, HookRule and RuleAction. They configure LLM and agent observability on a stack: evaluator definitions, online evaluation rules that sample and score live generations, and request-path guard rules that can redact or filter tools.

This is a candidate domain module, not stack-baseline content. Evaluators and rules depend on the workload being observed, so nothing here can be inferred from a stack request. The guard rules are the interesting part for a platform team, because they are a policy surface rather than a workload concern.

Decide whether this belongs in the reference at all before designing it. The README provider-family table already assigns unowned families to separate opt-in modules, and that may remain the right answer.

Verified: these five kinds are absent from the provider generated against Terraform v4.40.1 and are the whole Agent Observability upstream family.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A decision is recorded on whether Agent Observability is in scope for this reference, with the reasoning
- [ ] #2 If in scope, the API distinguishes platform-owned guard policy from workload-owned evaluators
- [ ] #3 If out of scope, the README provider-family table row states that and why
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Decision, taken by the repository owner: in scope as a full domain module

All five agento11y kinds behind an opt-in module, with the API distinguishing platform-owned guard
policy from workload-owned evaluators. HookRule and RuleAction are the request-path guards and are
platform policy; Collection, Evaluator and EvaluationRule are workload-owned and cannot be inferred
from a stack request.

The dependency on GCV-0010 is cleared: the pinned main build carries all five kinds. The README
provider-family table gained an agento11y row recording this decision and stating that no kind is
activated until the module ships.

Keep every rule body and evaluator definition inert and example-only. Real guard policy and real
evaluator definitions are environment-owned.
<!-- SECTION:NOTES:END -->
