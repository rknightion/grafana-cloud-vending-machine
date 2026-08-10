---
title: Installation
description: Bootstrapping Crossplane, External Secrets Operator, and the platform components, either directly or through Argo CD
---

# Installation

This page covers bootstrapping the platform itself. Once it is installed and healthy, vending a
stack is the copy-edit-review-commit path in [Getting started](getting-started.md).

Review every manifest and replace `platform.example.org`, the region, repository URLs, secret
paths, profiles, and the function package reference before treating either path below as
production. See [Configuration](configuration.md) for the full list of what to change.

## Direct installation

Suitable for a disposable or evaluation cluster.

### 1. Prepare the repository

Fork or copy the repository, choose an API group under a domain you control, and replace
`platform.example.org` everywhere. Change the repository URLs and function package path to your
fork. Keep `enabled/` empty until the controllers, provider, secret store, and organization
`ProviderConfig` are healthy.

### 2. Install Crossplane

```bash
helm upgrade --install crossplane crossplane \
  --repo https://charts.crossplane.io/stable \
  --version 2.3.4 \
  --namespace crossplane-system \
  --create-namespace \
  --values deploy/crossplane/values.yaml
```

Wait for the Crossplane and RBAC manager deployments to become `Available`.

### 3. Install External Secrets Operator

```bash
helm upgrade --install external-secrets external-secrets \
  --repo https://charts.external-secrets.io \
  --version 2.6.0 \
  --namespace external-secrets \
  --create-namespace \
  --values deploy/external-secrets/values.yaml
```

Configure workload identity before applying the `SecretStore`. Confirm the organization
credential exists at the configured remote path first — see [Secrets](secrets.md).

### 4. Install the platform and environment configuration

```bash
kubectl create namespace grafana-vending
kubectl apply -k platform
kubectl apply -k deploy/aws
```

`platform/` installs the Grafana provider (with signature verification), the vending
composition function (also signature-verified), the `ManagedResourceActivationPolicy`, the
XRDs and Compositions, and the extra composition RBAC needed for ESO's `PushSecret` and
`ExternalSecret` resources. `deploy/aws` installs the `SecretStore`, the organization
`ExternalSecret`, and the organization `ProviderConfig`.

Wait for the provider and function to become healthy:

```bash
kubectl wait provider.pkg.crossplane.io/provider-grafana \
  --for=condition=HealthyPackageRevision \
  --timeout=10m

kubectl wait function.pkg.crossplane.io/function-grafana-vending \
  --for=condition=HealthyPackageRevision \
  --timeout=10m

kubectl get providerconfig.grafana.m.crossplane.io \
  -n grafana-vending grafana-cloud-org
```

The optional profile secrets are intentionally excluded from `deploy/aws/kustomization.yaml`.
Apply `deploy/aws/optional-profile-secrets.yaml` only after the corresponding remote secrets and
profile definitions are ready — it contains OAuth inputs for the example `generic_oauth` and
`azuread` profiles plus the incident relay input. The example SAML profile uses public IdP
metadata and needs no committed key material.

### 5. Enable one request

See [Getting started](getting-started.md) for the full copy-edit-review-commit path. For a quick
evaluation:

```bash
cp -R examples/catalog/minimal enabled/my-stack
kubectl apply -k enabled/my-stack
```

### 6. Observe reconciliation

```bash
kubectl get grafanacloudstackrequests -n grafana-vending
kubectl describe grafanacloudstackrequest -n grafana-vending REPLACE_WITH_SLUG
kubectl get managed -n grafana-vending
kubectl get pushsecrets,externalsecrets -n grafana-vending
kubectl get providerconfigs.grafana.m.crossplane.io -n grafana-vending
```

## Argo CD installation

`deploy/argocd` contains four building blocks:

1. `crossplane.yaml` installs Crossplane 2.3.4 with the reference values.
2. `external-secrets.yaml` installs the version-gated ESO release.
3. `platform.yaml` installs `platform/` plus the environment-specific AWS `SecretStore` and
   `ProviderConfig`.
4. `requests-applicationset.yaml` creates one Argo CD `Application` per directory under
   `enabled/`.

The examples assume an Argo CD `AppProject` named `platform` and a repository Argo CD can read.
While a fork is private, configure repository access through Argo CD's credential mechanism
rather than placing credentials in an `Application`. A consumer of this public repository should
pin `targetRevision` to an audited commit SHA, not `main`, and update the pin only after
rendering and testing the new revision.

Apply the relevant settings from `deploy/argocd/argocd-values.yaml` to your Argo CD install:

- `application.resourceTrackingMethod: annotation` — prevents Crossplane-generated children from
  inheriting Argo CD ownership by label. This is not optional: without it, Argo CD's default
  label-based tracking claims resources Crossplane also considers its own.
- The `resource.customizations` health Lua scripts teach Argo CD to understand Crossplane
  package, managed-resource, and composite conditions, and to treat the
  `platform.example.org/*` custom resources as `Progressing` until their composed resources
  report `Ready`.
- `ProviderConfigUsage` is excluded from `resource.customizations` because it is an internal
  Crossplane bookkeeping object.

The Application sync waves order controllers before platform APIs and requests.
`SkipDryRunOnMissingResource=true` is needed while CRDs are still appearing.
`ServerSideApply=true` avoids client-side annotation-size limits on large CRDs.

### GitOps ownership and pruning

Argo CD `selfHeal` repairs changes to request objects and platform manifests. Crossplane repairs
changes to external Grafana resources according to its management policies. These are different
reconciliation loops — see [Architecture](architecture.md).

Argo CD pruning a `GrafanaCloudStackRequest` deletes the composite Kubernetes object and its
composed managed-resource objects. The managed resources omit the `Delete` management policy,
the `Stack` enables delete protection, rotating tokens set `deleteOnDestroy: false`, and
`PushSecret` uses `deletionPolicy: None`. External Grafana and secret-manager objects are
therefore **orphaned rather than destroyed**. This is intentional: a destructive decommission
must be a separately reviewed operation. See the decommission runbook in the project
[README](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook).

## Next steps

- [Getting started](getting-started.md) — vend your first stack.
- [Secrets](secrets.md) — the organization credential and per-stack token flow in detail.
- [Architecture](architecture.md) — the ownership boundaries between Argo CD, Crossplane, and
  ESO.
