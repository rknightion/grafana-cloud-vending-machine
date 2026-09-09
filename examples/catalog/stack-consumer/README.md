# Stack consumer

This inert example selects a platform-owned consumer profile and names an existing Grafana Cloud
stack. It demonstrates a consumer cluster using a stack that it does not vend locally: the request
contains the stack slug and region, while the Composition observes the provider-assigned stack ID
before it creates the profile-owned credential resources.

## Files

- `stack-consumer.yaml` selects the `example-reader` profile for the `exampleobservedstack` stack in
  `consumer-platform`.
- `kustomization.yaml` makes this directory directly renderable with Kustomize.

## Prerequisites

- The platform, Grafana provider, and vending function are healthy.
- The `example-reader` profile is installed in the enforced Composition, is bound to the
  `consumer-platform` namespace, and explicitly authorizes the selected stack slug and region.
- The platform-owned organization ProviderConfig and secret store referenced by that profile are
  healthy. The target Grafana Cloud stack already exists and is reachable through that profile.
- The consumer cluster does not hold a local `GrafanaCloudStackRequest` for the selected stack.

## Request boundary

The request only selects a profile and supplies the existing stack's slug and region. Provider
configuration, credential identity, scopes, token limits, and the generated output path belong to
the platform profile and are intentionally absent. The request cannot choose an organization,
provider configuration, credential, scope, lifetime, or output destination.

The composition first observes the existing stack with a non-creating managed resource and waits
for its provider-assigned ID. Once that identity is available, it renders the profile-owned access
policy, rotating token, and external credential publication resources. The catalog request itself
contains no credential data and no Kubernetes Secret manifest.

## Values to replace

- Replace `platform.example.org` with the API group used by the platform installation.
- Replace `example-reader` in both `metadata.name` and `spec.profile` with one approved,
  namespace-bound profile. These values must remain identical.
- Replace `consumer-platform` with the namespace authorized by that profile.
- Replace `exampleobservedstack` and `prod-us-central-0` with the existing stack slug and approved
  Grafana Cloud region authorized by the profile.
- Keep provider, credential, scope, limit, and output settings in the platform-owned profile rather
  than adding them to this request.

## Reconciliation

The catalog is not watched by Argo CD. Copy and adapt it into a deliberate live-request directory
only after reviewing the resulting render and the platform profile's authorization. Keep the
top-level `enabled/*` watch path unchanged and leave `enabled/` empty in this repository.

The observed stack is non-creating. Access-policy and rotating-token resources are owned by the
selected platform profile, and generated credentials are published through its fixed output
surface. Removing this request removes its consumer-side Kubernetes objects while retaining external
policies, tokens, and output documents. Tokens remain subject to their expiry. The consumer has no
Delete mode and does not transfer ownership of the existing Grafana Cloud stack.
