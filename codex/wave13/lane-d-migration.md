# GrafanaProvisioningRepository secure-value migration

## Breaking behavior

The released `GrafanaProvisioningRepository` API exposed
`spec.repository.secure.token`, `webhookSecret`, and `commitSigningKey`. This
wave keeps those field definitions for compatibility, but refuses the presence
of each field at admission and refuses the same paths in the renderer.

The refusal names a vendor secure-value reference defect. The defect is inferred
from the sibling `GrafanaProvisioningConnection` kind in the same vendor API
group; it was not observed on `Repository`. This repository does not probe a
live vendor stack for this behavior.

## Required consumer action

For every `GrafanaProvisioningRepository` claim:

1. Remove `spec.repository.secure.token`,
   `spec.repository.secure.webhookSecret`, and
   `spec.repository.secure.commitSigningKey` from the declarative source.
2. Do not replace them with a `create` form, a literal credential, or another
   secure-value reference. All three shapes are unsupported for this kind.
3. Keep `spec.repository.connectionRef.name` pointing at the existing
   `GrafanaProvisioningConnection` that supplies repository authentication.
4. Reapply the claim after removing the fields. Removing the fields is the
   migration; deleting the repository or its connection is not required.
5. If webhook or commit-signing material depended on one of the refused
   fields, leave that option unset until the vendor fixes reference-form
   handling and a follow-up release explicitly re-enables it.

The platform rejects name references, create forms, and literals on this released API. The vendor
evidence is narrower: refusal of the name-reference form is inferred from the sibling Connection
kind. A create form may be accepted by the vendor, but this API intentionally excludes it because
claim-authored credential creation is outside its security boundary.

An update that still contains any of the three fields is rejected, including a
name reference that was previously accepted by version 2.0.0. The existing
`!has(self.create)` validation remains in place, so credential creation from a
claim is still forbidden.
