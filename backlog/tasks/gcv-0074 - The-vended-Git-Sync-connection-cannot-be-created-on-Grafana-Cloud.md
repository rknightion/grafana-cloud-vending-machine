---
id: GCV-0074
title: The vended Git Sync connection cannot be created on Grafana Cloud
status: Parked
assignee: []
created_date: '2026-09-12 19:11'
updated_date: '2026-09-18 08:00'
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
- [x] #1 The reference form's 403 is reproduced in a test or recorded as an upstream defect with a link
- [x] #2 A decision is recorded on whether the create form's in-cluster credential exposure is accepted, and the XRD guards and docs are made consistent with it
- [ ] #3 GrafanaProvisioningConnection either reaches Ready against a real Grafana Cloud stack or is explicitly marked unusable in the catalog and request-schema docs
- [x] #4 The repository API documents the 300s sync.intervalSeconds floor and the fact that neither Grafana nor the provider reports it as drift
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-09-12 root fix, option 1 plus option 3's reporting half, decided by the owner.

DECISION (AC2). The renderer emits `secure.privateKey.create` from the ExternalSecret-materialised Kubernetes Secret, and the in-cluster exposure is accepted as a deliberate narrowing of the no-credential-literal rule scoped to this one API. Rejected: waiting on a secretRef-shaped field in provider-grafana, which blocks on someone else's release; and marking the API unusable, which leaves Git Sync unvendable and wave 11's headline false in practice.

The claim-side guards are UNCHANGED and still reject all three literal shapes on create and on update. The narrowing applies only to what the composition emits. Guards, catalog README and request-schema docs were made consistent in the same commit.

MECHANISM. `provisioningConnectionCredentialConfig` registers a `v1/Secret` requirement for `<name>-credential` in the request namespace and reads it through `request.GetRequiredResource`, the same mechanism `serviceBootstrapConfig` already uses at bootstrap.go:57, so no new RBAC. The Secret's identity is checked before the value is trusted: apiVersion, kind, name, namespace, absent deletionTimestamp, non-empty key.

The value is NEVER DECODED. A Kubernetes Secret `data` entry is already base64, which is exactly the encoding Grafana requires, so the plaintext PEM does not exist in function memory, in a function result, in a status field or in any log path. Decoding and re-encoding would be the only way to get it wrong.

The Connection is withheld when the credential has not materialised yet, rather than rendered without one - Grafana returns a clean 422 for a Connection with no private key, so a credential-less Connection is strictly worse than none.

The SecurevalueV1Beta1 is RETAINED on purpose and now duplicates the credential inside Grafana. It is the half of the chain Grafana accepts, so restoring the reference form once the 403 is fixed upstream is a one-line renderer change. `decrypters` therefore stays required and reviewed.

AC4 done: the 300s `sync.intervalSeconds` floor and the fact that neither Grafana nor the provider reports the override as drift are documented in the repository catalog README and the request-schema reference.

AC1 NOT DONE and not blocking: it asks for a test reproduction or an upstream link. The 403 is live-only, so it is not reproducible inside this repository's evidence boundary, and no upstream issue has been filed yet. The full live evidence is recorded here and in both docs. Filing the Grafana-side defect and adding its link is the remaining work.

AC3 NOT DONE and not blocking: whether the composed Connection now reaches Ready is live behaviour, which is not exercisable here. The fix is source-correct and unit-proven against the pinned CRD; the m7kni/portina-iac consumer must confirm it against a real stack. Resume by pinning the released tag there and reporting whether the Connection reaches Ready.

2026-09-12 AC1 AMENDED by owner decision: the Grafana-side defect will NOT be filed upstream, publicly or internally. AC1 as written asked for a test reproduction or an upstream link; neither is obtainable. The 403 is live-only, so it is outside this repository's evidence boundary and cannot be reproduced in a test, and with no filing there will be no link.

Recorded instead, and this is what AC1 now means: the reference form's 403 is documented in this repository as a vendor defect, with the full live evidence and the exact error string, at three places - this task's description, examples/catalog/provisioning-connection/README.md under 'Git Sync credential exposure', and the GrafanaProvisioningConnection entry in docs/reference/request-schema.md. The renderer itself carries the evidence in the doc comment on provisioningConnectionCredentialConfig, so the next person to touch that code finds out why the create form is there before they try to 'simplify' it back to a reference.

Consequence to carry forward: the create-form narrowing is now the permanent design, not a temporary workaround awaiting an upstream fix. The SecurevalueV1Beta1 is still rendered and still duplicates the credential inside Grafana. Revisit that only if Grafana's behaviour changes on its own, and re-verify live before removing anything - nothing in this repository will learn of such a change.

2026-09-18 AC3 researched against both live estates, as authorised. IT CANNOT BE SETTLED BY OBSERVATION YET, and the reason is concrete rather than a boundary argument: the fix is not deployed anywhere, and no Git Sync resource of any kind exists to observe.

WHAT WAS CHECKED, both clusters:
  ConnectionV0Alpha1          zero objects
  RepositoryV0Alpha1          zero objects
  SecurevalueV1Beta1          zero objects
So nothing has exercised this API since the throwaway probes were deleted on 2026-09-12.

AND THE DEPLOYED FUNCTION PREDATES THE FIX ON BOTH. platform/function/install.yaml pins digest sha256:50168c25dc02e98da3c16603919959f4a025f00aca685baa58e9616c566a29a8, which is the pin moved after the create-form fix landed. The estates run sha256:f4acdd026bed01b82e94e54e05411aabdba78258e7bac87a22fe7980aa817f8d and sha256:fb5e86a7a664572ef3383da16e85f1468c6d13ac8fd9abff61268daeb5bc44b8, the wave 11 digest and an older one. Neither contains the renderer that emits the create form, so even if a claim were applied today it would compose the refused reference shape again - and would resume the three-minute retry loop against the vendor that raised their alert the first time. Do not apply a Git Sync claim to either estate before the pin moves.

Provider versions also differ across the estates, v2.14.0 on one and a v2.13.0 build on the other, and only the former matches the repository pin.

RESUME BOUNDARY, in order, and each step is an estate action rather than repository work:
  1. roll the function package on the estate running provider v2.14.0 to the pinned digest sha256:50168c25dc02e98da3c16603919959f4a025f00aca685baa58e9616c566a29a8 and confirm the Function package reports Installed and Healthy at that digest
  2. apply one GrafanaProvisioningConnection claim with its ExternalSecret credential present
  3. report whether ConnectionV0Alpha1 reaches Ready, and whether status.health.healthy and status.token.expiration are set, which is what the working create-form probe produced within eight seconds on 2026-09-12
  4. if it does not reach Ready, capture the provider's message verbatim before deleting anything, because the object is the only evidence and deleting it to stop a vendor alert is what cost the verbatim record last time

Parked rather than Done because AC3 asks for one of two things and neither has happened: it has not reached Ready against a real stack, and it is not being marked unusable, since the source-level fix is correct and unit-proven. AC1, AC2 and AC4 remain satisfied as recorded above.

Do NOT close this by declaring live proof out of scope. That disposition is right for behaviour this repository can never reach, and this is not that: the estates exist, the authorisation exists, and the only missing step is a pin roll.
<!-- SECTION:NOTES:END -->
