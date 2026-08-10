---
title: Getting Started
description: Prerequisites and the copy-edit-review-commit path to vending your first Grafana Cloud stack
---

# Getting Started

!!! warning "`platform.example.org` is a documentation placeholder"
    Every request, XRD, and Composition in this repository uses the API group
    `platform.example.org`. It is not a production API group. Replace it everywhere — XRDs,
    Compositions, examples, Argo CD health customizations, and your own documentation — before
    treating a fork as production. See [Configuration](configuration.md) for the full list of
    places it appears.

## Prerequisites

Before you begin, you need:

- A Kubernetes cluster with **Crossplane pinned at 2.3.4** — required for namespaced composite
  resources, namespaced managed resources, and `ManagedResourceActivationPolicy`.
- **Argo CD**, with `application.resourceTrackingMethod: annotation` set (see
  [Installation](installation.md)) so Crossplane-generated children are not claimed by Argo's
  default label-based tracking.
- **External Secrets Operator pinned at 2.6.0** — chosen because 2.7.0 and 2.8.0 carry an open
  regression ([external-secrets/external-secrets#6593](https://github.com/external-secrets/external-secrets/issues/6593))
  that sends an empty replica-region request when creating an AWS Secrets Manager `PushSecret`
  target, which AWS rejects.
- An external secret store ESO can reach — the reference targets AWS Secrets Manager, but the
  vending function only emits `SecretStore` references, so another ESO provider works if it
  supports `ExternalSecret` and `PushSecret` with structured values.
- A **Grafana Cloud organization access policy token** with only the organization-level
  capabilities needed to manage stacks, stored in your secret manager (see
  [Secrets](secrets.md)).
- A Grafana Cloud **region slug** for the stack you intend to create (e.g.
  `prod-us-central-0`).

## Pinned versions

This reference pins versions and immutable artefacts rather than following latest tags:

| Component | Version | Why |
| --- | --- | --- |
| Crossplane | 2.3.4 | Namespaced composite resources, namespaced managed resources, `ManagedResourceActivationPolicy` |
| Grafana Crossplane provider | 2.13.0, immutable digest | Current provider release when this reference was published; generated from Grafana Terraform provider 4.40.0 |
| ESO Helm chart | 2.6.0 | Last release before the open AWS `PushSecret` creation regression in 2.7.0 and 2.8.0 |
| Cosign verification image | 3.1.2, immutable digest | Verifies the Grafana provider and this repository's function package |
| Composition function SDK | 0.7.1 | Pinned by the function Go module |
| Vending composition function | `sha256:673028c172bfeef3b1a9ab83ccb8b6db1320808b9f1e4b3c712aefc9ba4ed89b` | Signed amd64/arm64 package built from commit `fcd3bc5b57b1` |

The Grafana Crossplane provider describes itself as experimental and unsupported. It was
generated from Terraform provider 4.40.0 while a newer Terraform provider release exists and
contains fixes not yet in this Crossplane provider release — test provider upgrades against
non-production stacks before rollout. See [Installation](installation.md) for how the provider
and function packages are verified before Crossplane installs them.

## The copy-edit-review-commit path

Nothing under `examples/catalog` is applied by the supplied `ApplicationSet`, and the top-level
`enabled/` directory starts empty. Cloning or installing this platform cannot create a Grafana
Cloud stack on its own — a request has to be deliberately placed under `enabled/`.

Once the platform components are installed and healthy (see [Installation](installation.md)):

### 1. Choose a catalog example

Start from [`examples/catalog/minimal`](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/minimal)
for a safe baseline with rotating credentials, create-only content, and no SSO. See
[Reference → Catalog](reference/catalog.md) for every available example and what it demonstrates.

### 2. Copy it into `enabled/`

```bash
cp -R examples/catalog/minimal enabled/my-stack
```

### 3. Edit every placeholder

At minimum, replace:

- `metadata.name` and `spec.slug` — these must be identical, and Grafana Cloud stack slugs are
  globally unique.
- `spec.region` — a real Grafana Cloud region slug.
- `spec.usage` — a classification that also becomes part of the output secret path.
- `platform.example.org`, if you have forked the repository and repointed the API group.

The minimal example:

```yaml
apiVersion: platform.example.org/v1beta1
kind: GrafanaCloudStackRequest
metadata:
  name: replacewithunique01
  namespace: grafana-vending
spec:
  displayName: Example Grafana Cloud stack
  slug: replacewithunique01
  region: prod-us-central-0
  usage: development
  profile: standard
  baselineDashboards:
    enabled: true
  telemetryAccess:
    enabled: true
  plugins: []
  reconciliation:
    dashboards: createOnly
    homePreference: createOnly
  sso:
    mode: disabled
  monthlyReport:
    enabled: false
  incidentIntegration:
    enabled: false
```

See [Configuration](configuration.md) for what every field does.

### 4. Render and review

```bash
kubectl apply --dry-run=server -k enabled/my-stack
```

Check the server-side dry run output before committing anything real — this is the point at
which a copy-pasted example identity, an unintended SSO profile, or a stray plugin is cheapest to
catch.

### 5. Commit it

For GitOps, commit `enabled/my-stack` to the Git repository the `ApplicationSet` watches. The
`ApplicationSet` creates one Argo CD `Application` per directory under `enabled/*` and applies it.

For a disposable evaluation only, you can instead apply the directory directly:

```bash
kubectl apply -k enabled/my-stack
```

Do not put a real, patched request in this **public** repository's own `enabled/` directory — a
production deployment should keep enabled requests in a private GitOps repository. See
[Configuration → Public base, private environment overlay](configuration.md#public-base-private-environment-overlay).

### 6. Observe reconciliation

```bash
kubectl get grafanacloudstackrequests -n grafana-vending
kubectl describe grafanacloudstackrequest -n grafana-vending my-stack
kubectl get managed -n grafana-vending
kubectl get pushsecrets,externalsecrets -n grafana-vending
kubectl get providerconfigs.grafana.m.crossplane.io -n grafana-vending
```

The request becomes `Ready` only when every currently desired composed resource reports `Ready`.
Rotating-token resources render one reconciliation after Grafana assigns the parent
service-account or policy ID, so a brand-new request takes at least two reconciliation passes
before its credentials exist.

## Next steps

- [Installation](installation.md) — bootstrapping Crossplane, ESO, and the platform components.
- [Configuration](configuration.md) — the complete request API field reference.
- [Secrets](secrets.md) — how the organization credential gets in, and how per-stack tokens get
  out.
- [Architecture](architecture.md) — the three-controller split and reconciliation model.
