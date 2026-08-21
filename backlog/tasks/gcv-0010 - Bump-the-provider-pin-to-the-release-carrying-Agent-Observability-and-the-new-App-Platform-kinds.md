---
id: GCV-0010
title: >-
  Bump the provider pin to the release carrying Agent Observability and the new
  App Platform kinds
status: To Do
assignee: []
created_date: '2026-08-21 12:13'
labels: []
dependencies: []
references:
  - 'https://github.com/grafana/crossplane-provider-grafana/pull/658'
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The pinned Grafana Crossplane provider (v2.13.0) is generated against Terraform provider v4.40.1 and exposes 111 of the 121 upstream resources. The ten absent resources are the five Agent Observability kinds plus Alerting RoutingtreeV1Beta1 and RulesequenceV0Alpha1, oss QueryV1, cloud ComponentV1Alpha1, and assistant TermsAcceptance.

Cause: the provider's generator panics on an upstream resource category it does not recognise, and 'Agent Observability' was added upstream in Terraform provider v4.43.0 without a matching entry. An upstream PR adds that entry and regenerates; it is green but awaiting review. Once a provider release carries it, this repository moves its pin.

Moving the pin is not just a digest change. It touches the cosign verification identity, the Provider package reference, and the ManagedResourceActivationPolicy, which currently activates 19 kinds by name and must gain any newly emitted kind. The README provider-family table is the extension index and records the ownership decision per family, so a new family needs a row.

Do not activate kinds speculatively. Activation is driven by what the Compositions actually emit.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The provider package digest and the cosign --certificate-identity tag are updated together and the signature verification job passes
- [ ] #2 ManagedResourceActivationPolicy activates only kinds that a Composition emits after this change
- [ ] #3 The README provider-family table gains a row for the agento11y family stating its ownership decision
- [ ] #4 Known limitations records the provider version and the resource-count parity reached
- [ ] #5 ./scripts/validate.sh passes and the hosted validation run is green on the completing commit
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
