---
id: GCV-0081
title: >-
  GrafanaProvisioningRepository can only express the secure-value shape the
  vendor refuses, so setting any repository secret 403s forever
status: To Do
assignee: []
created_date: '2026-09-18 07:45'
labels:
  - needs-triage
  - vendor-defect
dependencies:
  - GCV-0074
documentation:
  - examples/catalog/provisioning-connection/README.md
priority: high
type: bug
ordinal: 81000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0074 established that the vendor refuses a provisioning secure value passed as a NAME REFERENCE, returning '403 PermissionDenied: identity type access-policy not allowed, expected either user or service-account: invalid identity' - an error about neither the caller nor an access policy. Only the inline create form, base64 encoded, is accepted. That was fixed for GrafanaProvisioningConnection by emitting the create form.

THE SIBLING API STILL ONLY SPEAKS THE REFUSED SHAPE. GrafanaProvisioningRepository exposes spec.repository.secure with token, webhookSecret and commitSigningKey, and every one of them is constrained to the reference form on both sides:

- the XRD pins each to an object requiring name, with a CEL rule '!has(self.create)' whose message reads 'secure values must reference an existing secure value by name; create is forbidden' (platform/apis/provisioning-v1beta1.yaml, the secure-value-name anchor);
- the renderer enforces the same thing and errors on anything else, 'provisioning repository secure values must be name references', rejecting a map that carries any key other than name (platform/function/provisioning.go, provisioningSecureValues).

So a consumer who sets any repository secret has no expressible shape that the vendor will accept, and no way to reach the one that works. Repository and Connection are the SAME vendor API group, provisioning.grafana.app/v0alpha1, and the 403 was traced to that group's secure-value resolution path rather than to the Connection kind, so the same refusal is expected here. That expectation is the thing to settle first: it is inference from a sibling kind, not an observation of this one.

WHY IT LOOKS PROVEN AND IS NOT. GCV-0074 records that the repository API 'vends correctly and reconciles clean'. That observation was made on a repository with NO secure values - authentication came from the connection - so it says nothing about this path. A green vend of the default path was read as a green vend of the kind.

THE CONSEQUENCE IS NOT CONFINED TO THIS PLATFORM, which is what raises the priority. The refused request is retried by Crossplane roughly every three minutes for as long as the claim exists. Each attempt makes the vendor's provisioning app fail to read the referenced secure value, and that is what raised an alert on the vendor's own on-call the first time this class was hit, on a weekend, on a live stack. A consumer who sets repository.secure.token today reproduces that. Nothing in this repository warns them.

WHAT MAKES THE FIX AWKWARD, so it is triage rather than a one-liner. The claim-side prohibition on credential literals is deliberate and the Connection fix kept it, narrowing only what the COMPOSITION emits: the value arrives through an ExternalSecret-materialised Kubernetes Secret and is passed base64 as Kubernetes already stores it, never decoded. The same mechanism is available here, but there are three secrets rather than one, so three requirements and three Secret keys, and the API is ALREADY RELEASED - spec.repository.secure shipped in a tagged version, so its field shapes are a frozen seam and a tightening is a breaking change under the repository's own rule.

Doing nothing is also an option worth costing: mark the three fields unusable, state the vendor defect and the on-call consequence in the request-schema and catalog docs, and refuse them at admission rather than letting a consumer generate vendor-side load. Refusing a released field is itself breaking.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Whether the vendor refuses repository.secure name references is settled against the pinned provider CRD and the vendor API group evidence, and the finding states plainly whether it is observed or inferred
- [ ] #2 A consumer setting any of token, webhookSecret or commitSigningKey either reaches a working repository or is refused at admission with a message naming the vendor defect; it is never left generating retries
- [ ] #3 If the create form is adopted, all three secrets go through the same ExternalSecret-materialised Secret mechanism as the connection credential, the value is never decoded inside the function, and the claim-side literal prohibition stays intact
- [ ] #4 The change is graded against the released-API rule: either it is additive, or it carries a breaking marker and a migration section
- [ ] #5 The vendor-side consequence of a permanently refused child - continuous retries against a live stack and an alert on the vendor's on-call - is documented where a consumer of this API will read it
- [ ] #6 A renderer test fails if any provisioning kind emits a secure-value name reference to the vendor
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
