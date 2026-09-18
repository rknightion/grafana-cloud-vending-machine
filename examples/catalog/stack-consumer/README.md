# Stack consumer

This inert catalog base selects the shipped `robk-telemetry-writer` profile for the `robk` stack in
`prod-gb-south-1`. It demonstrates a consumer cluster using a stack that it does not vend locally:
the request contains the stack slug and region, while the Composition observes the provider-assigned
stack ID before it creates the profile-owned credential resources.

## Files

- `stack-consumer.yaml` selects the `robk-telemetry-writer` profile for the `robk` stack in the
  `telemetry-consumers` namespace.
- `kustomization.yaml` makes this directory directly renderable with Kustomize.

## Prerequisites

- The platform, Grafana provider, and vending function are healthy.
- The `robk-telemetry-writer` profile is installed in the enforced Composition, is bound to the
  `telemetry-consumers` namespace, and explicitly authorizes the selected stack slug and region.
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

## Profile-owned output and scopes

The profile owns the fixed output path
`/platform/grafana-cloud/consumers/robk-telemetry`. Its PushSecret writes this `telemetry.json`
document there, with the vended token substituted during publication:

```json
{"stack_slug":"robk","stack_region":"prod-gb-south-1","access_policy_name":"robk-telemetry-access-policy","access_policy_token":"<vended token>"}
```

The policy grants only `metrics:write`, `logs:write`, and `traces:write`: each permits one telemetry
signal this consumer publishes. It does not grant `stacks:read`, because this credential does not
query stack inventory, and it does not add any scope for a signal it does not publish.

The consuming estate must read the published value from its own approved secret store. Live secret-store
read-back is not exercised, and not exercisable here.

## Adaptation boundary

This public profile is deliberately a working reference for the named target. A different consumer
estate must define and review a separate profile in its own platform configuration; changing this
request cannot select a different provider, credential, scope, lifetime, or output destination. In
all cases, `metadata.name` and `spec.profile` must remain identical.

## Reconciliation

The catalog is not watched by Argo CD. Copy and adapt it into a deliberate live-request directory
only after reviewing the resulting render and the platform profile's authorization. Keep the
top-level `enabled/*` watch path unchanged and leave `enabled/` empty in this repository.

The observed stack is non-creating. Access-policy and rotating-token resources are owned by the
selected platform profile, and generated credentials are published through its fixed output
surface. Removing this request removes its consumer-side Kubernetes objects while retaining external
policies, tokens, and output documents. Tokens remain subject to their expiry. The consumer has no
Delete mode and does not transfer ownership of the existing Grafana Cloud stack.

`AccessPolicyToken` rotation is not implemented while GCV-0075 remains Parked, so this example is
not self-renewing. Before the 2026-10-09 early-rotation window, complete an approved credential
replacement, publish it, and verify that the consuming estate has reloaded it. If that handover
cannot be completed and verified, remove the request before the credential expires.

The old `example-reader` placeholder was removed rather than retained as documentation. A live
reference profile and this renderable inert catalog make the supported request unambiguous; a second
placeholder would be denied by profile authorization and could not be used safely as an alternative.
