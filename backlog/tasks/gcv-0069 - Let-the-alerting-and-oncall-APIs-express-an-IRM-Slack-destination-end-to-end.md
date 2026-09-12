---
id: GCV-0069
title: Let the alerting and oncall APIs express an IRM Slack destination end to end
status: Parked
assignee:
  - '@codex'
created_date: '2026-09-11 13:22'
updated_date: '2026-09-12 13:08'
labels: []
dependencies: []
ordinal: 69000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A consumer repository needed four things wired from one claim set and only the fourth was expressible, so it wired the other three directly against the stack API instead. That is the gap.

WHAT IS MISSING, field by field.

1. INTEGRATION TYPE IS HARDCODED. The oncall API's composite carries no integration-type field at all, and the composition function writes the type as a literal inbound-email integration. A consumer that wants Grafana Alerting to raise IRM alert groups needs the grafana_alerting integration type, which cannot be selected. Needs an enum field on the composite spec plus a branch in the composition function where the literal is written today.

2. ESCALATION CHAINS CANNOT CARRY A CHAT-CHANNEL STEP. The escalation block on the composite accepts only a final-step discriminator and a schedule reference, and CEL validation pins both to one value each. The composition emits exactly one step, notify-on-call-from-schedule. There is no step array and no notify-target field of any kind, so a chain whose only destination is a chat channel is not expressible.

   Worth carrying upstream: the vendor's own public escalation-policy API has no chat-channel step type either. In that product's model the chat channel is a property of the integration's ROUTE, not of the escalation chain. So the field to add is a route-level channel reference on the oncall composite, not a new escalation step - and it takes the channel's opaque id, not its display name. Verified live: posting a route create with a display name in the channel-id field returns 200 and SILENTLY DROPS the channel, leaving a route with no destination and no error.

3. CONTACT POINTS COLLAPSE TO EMAIL. The routing composite's contact-point entry can reference an oncall composite, but the renderer resolves that reference to the observed inbound-email address and emits an email-type contact point. The vendor's own IRM contact-point type takes a required integration URL. So a contact point of the IRM type, pointing at an integration, is not expressible - only an email bridge to an inbound-email integration is.

   NOTE THE SECRET PROBLEM THIS CREATES for the consumer, because it is the reason this cannot simply be worked around in the consumer repo: the IRM integration URL is a bearer-equivalent webhook and the vendor marks that contact-point option non-secure, so it cannot be committed to a git-tracked manifest. A composition that resolves the URL from the observed integration inside the control plane is strictly better than any consumer-side arrangement, because the value never leaves it.

4. THE NOTIFICATION POLICY ROUTE IS ALREADY EXPRESSIBLE and needs no change. The routing composite models the root policy contact point plus a flat list of first-level child routes with matchers, and the matcher operator enum already includes the regex form. It replaces the whole provider-managed tree rather than merging, which is documented on the composite and is the correct behaviour.

SCOPE NOTE. Items 1 to 3 are one coherent change: an integration type, a route-level channel reference, and a contact point that carries an integration URL. Splitting them ships a composition that can create an integration nothing can route to.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The oncall composite accepts an integration type covering at least the Grafana Alerting type, and the composition writes it instead of a literal
- [ ] #2 The oncall composite accepts a route-level chat-channel reference taking the channel's opaque id, and the composition sets it on the route
- [ ] #3 A chat-channel reference that the control plane cannot resolve fails the claim loudly rather than reconciling a route with no destination
- [ ] #4 The routing composite can emit a contact point of the IRM type whose integration URL is resolved inside the control plane from the referenced oncall composite, never passed in by the consumer
- [ ] #5 The four shapes are provable together from one claim set: integration, escalation chain, IRM-typed contact point, and a regex-matched notification policy route
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Lane D adds selectable integration type, exactly-one Slack channel reference or opaque id, unresolved-reference readiness blocking, and the IRM contact-point cross-resource reference.
2. Prove the four-resource route in one claim set while preserving the existing regex notification policy surface.
3. Root wires SlackChannel activation and mapping, runs the integrated gate and security review, then records exact-SHA hosted evidence.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-09-12 AC2 amendment: the route accepts exactly one of channelRef.name, resolved through an observe-only SlackChannel, or channelId as an opaque-id escape hatch. The pinned provider supports slackChannelRef with Required resolution; passing a display name as channelId was observed to succeed while silently dropping the destination, so a reference-by-name path is required.

Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 10 root-only blocker: the required control for integrations.oncall.grafana.m.crossplane.io failed. Canonical normalized package hash 6038ad9a805467cb372ebbf38135c09686d6ce35f54918aafbeb13747c713f36 differs from fixture hash 83c7e3627bb8f51c9167412c24881cd055cab852fa96841d3b7ed04339306620; the sole structural difference is top-level $.status present in the cached package CRD and absent from the fixture. No fixture extraction or lane dispatch occurred. Resume only after independently verifying the cached package layer and reconciling the fixture provenance contract, then rerun the equality control.
<!-- SECTION:FINAL_SUMMARY:END -->
