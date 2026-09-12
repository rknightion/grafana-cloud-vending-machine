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

## Files

- `connection.yaml` is a deliberately inert request shape.
- `kustomization.yaml` renders the example as a Kustomize base.

## Values to replace

Replace the platform API group, stack name, Git URL, secret-store key and property, decrypter identity, and connection metadata. The GitHub App ID and installation ID are placeholders only. Their correct live identity is not exercised and cannot be exercised in this public reference.
