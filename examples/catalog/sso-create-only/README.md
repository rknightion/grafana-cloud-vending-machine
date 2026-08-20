# Create-only OAuth SSO

Use this example to initialize an OAuth/OIDC integration from platform policy and then hand ownership of later SSO changes to stack administrators. It omits `spec.lifecycle.externalResources`, so external-resource handling remains the safe `Retain` default.

## Prerequisites

- The minimal platform and organization ProviderConfig prerequisites are met.
- The Composition contains an approved `example-oidc` profile adapted to the real identity provider.
- The profile's client secret is materialized into its referenced Kubernetes Secret through the external secret manager.
- A tested break-glass or alternate administrator login exists before SSO is enabled.

## Values to replace

- Replace `platform.example.org`, every `replacewithunique03` occurrence, the region, usage, profile, and display name. In the reference, `usage` must be the immutable platform vocabulary `development` or `production` and forms part of `{outputSecretPrefix}/{region}/{usage}/{slug}`.
- Replace `example-oidc` with an approved Composition profile name; do not place OAuth endpoints or client secrets in the request.

With the default `Retain` lifecycle, removing this request orphans external resources. A platform-
authorized Delete decommission requires three reviewed stages: first arm
`spec.lifecycle.externalResources: Delete` and wait for `status.deletionReady=true`; then remove
dependent access claims and wait for their Kubernetes objects and finalizers to be gone while the
Stack still exists; finally remove the request. Arming alone never deletes anything.

## Reconciliation

The `createOnly` mode supplies the approved SSO settings only during creation. Crossplane continues to observe the managed resource but does not repair later administrator changes to those settings. Switch deliberately to `enforced` if Git should become authoritative again, or to `observeOnly` for an explicit read-only ownership model.
