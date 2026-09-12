---
id: GCV-0074
title: The vended Git Sync connection cannot be created on Grafana Cloud
status: To Do
assignee: []
created_date: '2026-09-12 19:11'
labels:
  - needs-triage
dependencies: []
type: bug
ordinal: 74000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GrafanaProvisioningConnection composes a ConnectionV0Alpha1 whose forProvider carries secure.privateKey: {name: <securevalue>}, the reference form. Grafana Cloud refuses that form outright with HTTP 403, so no vended Git Sync connection can ever reach Ready and the separately vended GrafanaProvisioningRepository sits unhealthy for want of one.

VERIFIED LIVE on 2026-09-12 against stack portina (1824620, prod-gb-south-1) from the m7kni/portina-iac consumer, at pin 3d789c8.

THE ERROR NAMES THE WRONG THING, which is the expensive part:

  403 PermissionDenied: identity type access-policy not allowed, expected either user or service-account: invalid identity

It is about neither the caller nor an access policy. The identical request fails from the stack's own glsa_ service-account token (service-account:20, 'Grafana vending controller', which /api/user confirms is a service account, not an access policy) AND from a real interactive user via gcx. It also fails whoever created the referenced secure value: a secure value created by the vending controller and one created by a user, both with decrypters [provisioning.grafana.app], are refused the same way.

WHAT DOES WORK is the create form with a BASE64-ENCODED PEM:

  secure: {privateKey: {create: <base64 of the PEM>}}

A throwaway connection built that way reached status.health.healthy true and minted a GitHub App installation token within eight seconds. A raw PEM in the same field is rejected with 'privateKey must be base64 encoded', so the base64 is required rather than incidental.

TWO CHEAP DIAGNOSTIC FACTS for whoever picks this up:

- Validation runs BEFORE the 403. Omitting secure entirely returns a clean 422 'secure.privateKey: Required value: privateKey must be specified for GitHub connection', which is how you tell a payload problem from this one.
- ?dryRun=All is IGNORED by secret.grafana.app: two secure values posted with it persisted as real objects and had to be deleted. There is no non-mutating way to probe these APIs. kubectl apply --dry-run=server against the consumer cluster IS honoured, so the two behave oppositely.

THE FIX COLLIDES WITH THIS PACKAGE'S OWN SECURITY RULE, which is why this is triage rather than a one-line change. The create form needs the credential as a literal in forProvider, so it would appear in the composed managed resource and be readable by anyone with get on the claim namespace - exactly what the XRD's credential.inline, credential.create and spec.secure guards exist to prevent. provider-grafana offers no alternative: connectionv0alpha1s.oss.grafana.m.crossplane.io has no secretRef-shaped field anywhere in its schema, only the secure.privateKey string map. Putting it in initProvider is not an escape either, because the common management policies include LateInitialize and would promote it into forProvider on the first reconcile after create.

So the options each need a decision rather than an implementation:
  1. emit the create form from the ExternalSecret-materialised value and accept the in-cluster exposure, documenting it as a deliberate narrowing of the no-literal rule for this one API
  2. get a secretRef-shaped secure field into provider-grafana first and depend on it
  3. treat the reference form's 403 as a Grafana Cloud defect, raise it upstream, and park this API until it works

GrafanaProvisioningRepository is unaffected and vends correctly: a repository with sync.target folderless, workflows [write] and a github path landed with the declared uid and reconciles clean. Separately worth recording in the repository API's docs: Grafana Cloud's floor for sync.intervalSeconds is 300 and it silently raises anything lower, while the Crossplane provider reads back its own requested 60 and reports no drift, so neither side flags the override.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The reference form's 403 is reproduced in a test or recorded as an upstream defect with a link
- [ ] #2 A decision is recorded on whether the create form's in-cluster credential exposure is accepted, and the XRD guards and docs are made consistent with it
- [ ] #3 GrafanaProvisioningConnection either reaches Ready against a real Grafana Cloud stack or is explicitly marked unusable in the catalog and request-schema docs
- [ ] #4 The repository API documents the 300s sync.intervalSeconds floor and the fact that neither Grafana nor the provider reports it as drift
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
