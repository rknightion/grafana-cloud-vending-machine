# Git provisioning connection

Apply this connection before a `GrafanaProvisioningRepository` that references it. The repository only names an existing Grafana connection; this API establishes that connection and its credential bridge first.

The request contains no credential literal, Kubernetes Secret value, or provider `secure` create map, and those guards reject all three on create and on update. Its `credential.remoteRef` identifies a value in the configured consumer secret store. An ExternalSecret materializes that value into a Kubernetes Secret in the request namespace, and a Grafana `SecurevalueV1Beta1` references that Secret.

## Git Sync credential exposure

**The composed Grafana Connection carries the private key as a base64 literal in its `forProvider`, and this is a deliberate narrowing of this repository's no-credential-literal rule, scoped to this one API.** Read this before applying the catalog.

Grafana Cloud refuses a Connection whose `secure.privateKey` is the reference form `{name: <securevalue>}`, returning `403 PermissionDenied: identity type access-policy not allowed, expected either user or service-account`. That error names neither the caller nor an access policy: it was reproduced identically from a stack service-account token and from an interactive user, against secure values created by each. Only `secure.privateKey.create` is accepted, and only base64-encoded. The pinned provider exposes no `secretRef`-shaped field on this kind, and `initProvider` is not an escape because `LateInitialize` in the common management policies promotes the value into `forProvider` on the first reconcile after create.

What that costs, precisely:

- The key is readable by any identity with `get` on the composed `ConnectionV0Alpha1` in the request namespace. The ExternalSecret also materializes the same key into a Kubernetes Secret there, but **Secret and managed-resource RBAC are granted independently**: a role that can read `ConnectionV0Alpha1` need not be able to read Secrets, so this is a second reader set and it can widen who holds the credential, not merely restate the existing one. **Audit and restrict `get` on `connectionv0alpha1s.oss.grafana.m.crossplane.io` in the request namespace before deploying this catalog**, alongside Secret access.
- The key is never decoded inside the composition function. A Kubernetes Secret's `data` entry is already base64, which is exactly the encoding Grafana requires, so the plaintext PEM does not exist in function memory, in a function result, in a status field or in any log path.
- **No claim field accepts the credential.** The narrowing applies to what the composition emits, not to what a consumer may write, so a credential still never enters a manifest or Git.
- The `SecurevalueV1Beta1` is still created and now duplicates the credential inside Grafana. It is retained on purpose: it is the half of the chain Grafana accepts, so restoring the reference form once the 403 is fixed upstream is a one-line renderer change. Its `decrypters` entry is therefore still required and still reviewed.

The 403 is recorded as a suspected Grafana Cloud defect in GCV-0074 and is expected to be raised upstream. When it is fixed, this narrowing is reverted and this section goes with it.

`decrypters` requires exactly one entry. Set it to the reviewed Grafana identity allowed to read the secure value; it is not inferred by this catalog.

Removing this claim removes the ExternalSecret and its owner-managed Kubernetes Secret, but the common non-destructive management policies retain the external Grafana Connection and Securevalue. Ordered external decommission is a separate, platform-authorized lifecycle decision; do not switch these children to `Delete` independently.

## Decommissioning a Git Sync credential

`spec.lifecycle.externalResources` defaults to `Retain`: the external Connection and Securevalue stay in Grafana when the claim is removed. `Delete` is an exceptional, platform-authorized path. The Composition's platform-owned `deletionAuthorizations` must bind the request namespace, name and UID to the fixed provisioning-connection profile before a claim can select it. A claim cannot select that profile itself.

The deletion path remains two reviewable changes. First, set `spec.lifecycle.externalResources: Delete` and wait for `status.decommission.phase: Armed`. `Armed` requires the observed Connection to carry Delete management policy, its deterministic external name, and the Crossplane managed-resource finalizer. Then remove the claim. The function first persists `ConnectionDeleting`, then withdraws the Connection on the next reconciliation. It follows the same two-step witness for the Securevalue: `SecurevalueArming`, then `SecurevalueDeleting`, then withdrawal. Deleting the Connection first stops Git Sync from using the credential before the Securevalue copy is revoked.

`status.decommission.phase` is the durable completion signal. It progresses through `PreparingConnection`, `Armed`, `ConnectionDeleting`, `SecurevalueArming`, `SecurevalueDeleting`, and `Complete`; `status.decommission.complete: true` is emitted only after reconciliation has observed neither external child and the preceding `SecurevalueDeleting` status witness. A missing child without its witness is reported as unproven, not complete. Use this status rather than inferring completion from the disappearance of composed objects.

The ExternalSecret's target has `creationPolicy: Owner` and `deletionPolicy: Retain`. Its owner lifecycle removes the Kubernetes Secret when the ExternalSecret is removed with the claim. `deletionPolicy: Retain` only preserves the last synced Secret if the remote source disappears during normal reconciliation. The source secret-store value has its own lifecycle: this API does not delete or revoke it. Revoke or rotate that source under the secret-store owner's process; the ordered path revokes the two Grafana copies and their Git Sync use.

## Repository secure-value migration

Version 2.0.0 accepted `spec.repository.secure.token`, `webhookSecret`, and `commitSigningKey` on a
`GrafanaProvisioningRepository`. The current API keeps those field definitions but rejects each
field at admission and in the renderer. An update that still contains one is rejected, including a
name reference that version 2.0.0 accepted.

This refusal is based on the `GrafanaProvisioningConnection` result above. The sibling kind uses the
same vendor API group, but the repository path has not been exercised. Probing it would create live
resources because the vendor secret API ignores `dryRun`.

Remove all three fields from the repository claim and keep
`spec.repository.connectionRef.name` pointing at the existing connection. Do not replace them with
a `create` form, credential literal, or another secure-value reference. Reapply the claim after
removing the fields; the repository and its connection do not need to be deleted. Leave webhook and
commit-signing material unset until a later release explicitly re-enables the reference form.

## Files

- `connection.yaml` is a deliberately inert request shape.
- `kustomization.yaml` renders the example as a Kustomize base.

## Values to replace

Replace the platform API group, stack name, Git URL, secret-store key and property, decrypter identity, and connection metadata. The GitHub App ID and installation ID are placeholders only. Their correct live identity is not exercised and cannot be exercised in this public reference.
