# Asserts module

This inert catalog example selects the platform-owned `baseline` Asserts profile for one
referenced stack. The request is profile-only: it supplies `spec.stackRef.name` and
`spec.profile`, while the Composition supplies provider settings, resource names, credential
references, and finite platform-owned caps.

The profile covers the nine namespaced Asserts kinds:

- `CustomModelRules`
- `LogConfig`
- `NotificationAlertsConfig`
- `ProfileConfig`
- `PromRuleFile`
- `Stack`
- `SuppressedAssertionsConfig`
- `Thresholds`
- `TraceConfig`

The cluster-scoped duplicate of each kind is excluded. The example contains no Secret, token,
endpoint, or provider configuration.

## Files

- `asserts.yaml` selects the profile for a same-name stack reference.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Prerequisites

- The platform, Grafana provider, and vending function are healthy.
- A Ready `GrafanaCloudStackRequest` named by `spec.stackRef.name` exists in this request's
  namespace.
- The corresponding per-stack `ProviderConfig` exists and is healthy in the same namespace.
- The environment has verified the Asserts entitlement for the referenced stack.
- The platform profile's required cloud-access-policy token Secret reference resolves. The
  optional Grafana token Secret reference must also resolve when the profile uses it. Those
  Secret objects and references are environment and platform-profile inputs, not catalog
  resources.

## Values to replace

- Replace `platform.example.org` with the API group used by the target platform.
- Replace both `exampleasserts01` values with the name of the Ready stack request. The request
  name and `spec.stackRef.name` must remain identical so one composite owns the stack's complete
  Asserts surface.
- Replace `grafana-vending` when the target namespace differs, and select only a profile approved
  by the platform.

## Reconciliation

The composition waits for the referenced stack to expose its provider-observed Cloud Stack ID
before emitting Asserts children. It never derives a Stack identity from a slug. Named
configurations use their profile names as external names; the whole Thresholds set is one
singleton with the fixed external name `custom_thresholds`.

The `GrafanaAsserts` composite is the sole declarative owner for this stack's Asserts surface.
Profile changes remain platform-controlled. If a profile or prerequisite transition would
withdraw an already observed child, reconciliation refuses that transition and retains the
existing child until an explicit decommission is performed.

This directory is reference material and is not watched by the example ApplicationSet.

When the profile enables Stack onboarding it declares a non-empty dataset list. The baseline uses
one explicit Prometheus dataset; automatic dataset discovery is refused because its result cannot
be bounded by the platform profile. Entitlement and data-source availability remain prerequisites.
