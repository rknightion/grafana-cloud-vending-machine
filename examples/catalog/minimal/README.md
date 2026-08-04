# Minimal stack

Use this example for the smallest production-oriented vending request. It creates a deletion-protected Grafana Cloud stack, rotating administrator and telemetry credentials, External Secrets Operator publication resources, and the three starter folders and dashboards.

## Files

- `stack.yaml` declares the `GrafanaCloudStackRequest`.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Prerequisites

- The platform, Grafana provider, vending function, and External Secrets Operator are healthy.
- The `grafana-cloud-org` organization ProviderConfig and `grafana-vending-secrets` SecretStore exist, or the Composition input has been adapted to your names.
- The organization credential can create stacks in the selected Grafana Cloud region.

## Values to replace

- Replace `platform.example.org` with your API group.
- Replace every `replacewithunique01` occurrence with one globally unique stack slug. Keep `metadata.name` and `spec.slug` identical.
- Replace `prod-us-central-0`, `usage`, `profile`, and the display name with approved environment values.
- Review whether telemetry credentials and baseline dashboards should be enabled.

## Reconciliation

The stack and credential resources are continuously reconciled. Dashboard JSON and the home-dashboard preference use `createOnly`, so the platform initializes them but preserves later administrator edits. Folder titles and other managed fields remain reconciled. SSO, reports, plugins, and incident integration are disabled in this example.
