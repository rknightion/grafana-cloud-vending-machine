# Git provisioning connection

Apply this connection before a `GrafanaProvisioningRepository` that references it. The repository only names an existing Grafana connection; this API establishes that connection and its credential bridge first.

The request contains no credential literal, Kubernetes Secret value, or provider `secure` create map. Its `credential.remoteRef` identifies a value in the configured consumer secret store. An ExternalSecret materializes that value in the request namespace, then a Grafana `SecurevalueV1Beta1` references it, and finally the Grafana connection references that secure value.

`decrypters` requires exactly one entry. Set it to the reviewed Grafana identity allowed to read the secure value; it is not inferred by this catalog.

Removing this claim removes the ExternalSecret and its owner-managed Kubernetes Secret, but the common non-destructive management policies retain the external Grafana Connection and Securevalue. Ordered external decommission is a separate, platform-authorized lifecycle decision; do not switch these children to `Delete` independently.

## Files

- `connection.yaml` is a deliberately inert request shape.
- `kustomization.yaml` renders the example as a Kustomize base.

## Values to replace

Replace the platform API group, stack name, Git URL, secret-store key and property, decrypter identity, and connection metadata. The GitHub App ID and installation ID are placeholders only. Their correct live identity is not exercised and cannot be exercised in this public reference.
