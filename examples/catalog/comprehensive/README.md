# Comprehensive stack and access model

Use this example to evaluate the complete public vending API. It intentionally exercises every custom kind and most optional stack features; it is a coverage reference, not a recommendation to enable every feature for every stack. It omits `spec.lifecycle.externalResources`, so external-resource handling remains the safe `Retain` default.

## Files

- `stack.yaml` enables rotating credentials, starter content, a plugin, enforced OAuth SSO, a monthly report, and incident relay resources.
- `custom-role.yaml` shows the compact one-Team, one-custom-Role binding API.
- `team-access.yaml` shows direct and synchronized membership, Team preferences, multiple custom roles, and a fixed-role assignment.
- `content-access.yaml` owns the complete ACLs for one starter Folder and Dashboard.
- `kustomization.yaml` renders all four manifests together.

## Prerequisites

- All minimal-example prerequisites are met.
- The Composition contains approved `example-oidc` and `example-relay` profiles adapted to real endpoints, and their referenced Secrets are populated through your secret manager.
- The target account has the required reporting, plugin, Team Sync, RBAC, OnCall, and Alerting capabilities.
- Direct Team members already exist in Grafana; external group identifiers match the IdP; every fixed-role UID is inventoried from the target stack.
- Custom-role actions and scopes are valid for the target Grafana version, and report recipients and relay destinations have been approved.

## Values to replace

- Replace the API group and every `replacewithunique02` occurrence, keeping the request name and slug identical.
- Replace the region, usage, profile, display name, change reference, and configuration-item reference. In the reference, `usage` must be the immutable platform vocabulary `development` or `production` and forms part of `{outputSecretPrefix}/{region}/{usage}/{slug}`.
- Replace SSO and incident profile names, plugin/version, report addresses, Team names, email addresses, members, external groups, custom-role UIDs/actions/scopes, fixed-role UIDs, and ACL subjects or targets.
- Remove any optional feature whose prerequisite or entitlement is unavailable.

## Reconciliation

Dashboard JSON, home preference, and SSO are enforced, so out-of-band edits are restored. The incident template is initialized with `createOnly`; later template edits are preserved while its managed endpoint contract remains reconciled. Team properties, direct members, role assignments, custom roles, and declared ACLs are enforced. Each content policy owns the complete ACL for its target, so omitted grants are removed during reconciliation.

The default `Retain` lifecycle means removing this request orphans external resources. An authorized
Delete decommission still requires three reviewed stages: first set
`spec.lifecycle.externalResources: Delete` and wait for `status.deletionReady=true`; then remove the
dependent access claims and wait for their Kubernetes objects and finalizers to be gone while the
Stack still exists; finally remove the request. The first stage arms intent only.
