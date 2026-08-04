# Create-only OAuth SSO

Use this example to initialize an OAuth/OIDC integration from platform policy and then hand ownership of later SSO changes to stack administrators.

## Prerequisites

- The minimal platform and organization ProviderConfig prerequisites are met.
- The Composition contains an approved `example-oidc` profile adapted to the real identity provider.
- The profile's client secret is materialized into its referenced Kubernetes Secret through the external secret manager.
- A tested break-glass or alternate administrator login exists before SSO is enabled.

## Values to replace

- Replace `platform.example.org`, every `replacewithunique03` occurrence, the region, usage, profile, and display name.
- Replace `example-oidc` with an approved Composition profile name; do not place OAuth endpoints or client secrets in the request.

## Reconciliation

The `createOnly` mode supplies the approved SSO settings only during creation. Crossplane continues to observe the managed resource but does not repair later administrator changes to those settings. Switch deliberately to `enforced` if Git should become authoritative again, or to `observeOnly` for an explicit read-only ownership model.
