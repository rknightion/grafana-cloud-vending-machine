# Access and RBAC

Use this example when stack provisioning must include a practical Grafana access model: direct and IdP-synchronized Team membership, Team preferences, a least-privilege custom role, a Grafana fixed role, and complete Folder and Dashboard ACLs. It omits `spec.lifecycle.externalResources`, so external-resource handling remains the safe `Retain` default.

## Files

- `stack.yaml` creates the target stack and selects enforced OAuth SSO.
- `team-access.yaml` owns the Team, membership sources, preferences, and additive role assignments.
- `content-access.yaml` owns the complete ACL for one starter Folder and the home Dashboard.

## Prerequisites

- The minimal stack prerequisites are met.
- The Composition has an approved `example-oidc` profile and its client secret is materialized from the external secret manager.
- Direct members already exist in Grafana, Team Sync is available, and external group identifiers match the IdP.
- Every fixed-role UID has been inventoried from the target stack. Role availability can vary by Grafana version, edition, entitlement, and stack age.
- The custom-role actions and scopes have been reviewed against the target Grafana version.

## Values to replace

- Replace the API group and every `replacewithunique04` occurrence, keeping `metadata.name` and `spec.slug` identical.
- Replace the region, usage, profile, display name, SSO profile, Team details, members, external groups, role names/UIDs/actions/scopes, fixed-role UIDs, and ACL grants or targets. In the reference, `usage` must be the immutable platform vocabulary `development` or `production` and forms part of `{outputSecretPrefix}/{region}/{usage}/{slug}`.
- Ensure each Folder or Dashboard is owned by exactly one `GrafanaContentAccessPolicy`.

## Reconciliation

SSO is enforced, so Crossplane restores out-of-band changes. Dashboard JSON and the home preference are `createOnly`. Team properties, direct members, preferences, custom roles, and role assignments are enforced; externally synchronized members are ignored by direct-membership reconciliation. Each content policy is authoritative for its target ACL and restores the complete declared grant set.

With the default `Retain` lifecycle, deleting this request orphans external resources. A platform-
authorized Delete decommission has three reviewed stages: arm
`spec.lifecycle.externalResources: Delete` and wait for `status.deletionReady=true`; remove these
dependent access claims and wait for their Kubernetes objects and finalizers to be gone while the
Stack still exists; then remove the request. Arming does not delete anything by itself.
