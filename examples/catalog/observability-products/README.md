# Observability product activation

This inert example enables Application Observability on one example stack. The
`spec.products` fields are activation toggles only:

- `applicationObservability` emits the provider's Application Observability singleton.
- `kubernetesObservability` emits the provider's Kubernetes Observability singleton.
- `databaseObservability` emits the provider's Database Observability singleton.

Each enabled toggle emits one cloud-family singleton with the fixed external
name `global`. Omitting a toggle or setting it to `false` emits no singleton.
When a previously enabled toggle is disabled, Crossplane will delete the
previously desired singleton from the composed resource set under its management
policy.
The standard policy omits external `Delete`, so this withdrawal does not request
deletion of the external product configuration.

These fields do not configure the products. Kubernetes Monitoring configuration
lives in Helm chart values, while Application and Database Observability
configuration belongs in their respective onboarding flows. Those configuration
surfaces are outside this stack-request provider.

Replace the example identity and platform references before using this manifest
in a consumer catalog. This directory is not watched by the example
ApplicationSet.
