---
title: Getting Started
description: Prerequisites and the copy-edit-review-commit path to vending your first Grafana Cloud stack
---

# Getting Started

!!! warning "`platform.example.org` is a documentation placeholder"
    Every request, XRD, and Composition in this repository uses the API group
    `platform.example.org`. It is not a production API group. Replace it everywhere — XRDs,
    Compositions, examples, Argo CD health customizations, and your own documentation — before
    treating a fork as production. See [Configuration](configuration.md) for platform policy and
    [Request Schema Reference](reference/request-schema.md) for the per-API fields.

## Prerequisites

Before you begin, you need:

- A cluster running **Kubernetes 1.30 or later** - required by the cluster-scoped
  `ValidatingAdmissionPolicy` resources installed with the platform. Keep the cloud integrations,
  k6, ML, PDC, and service-account XRDs paired with their named Compositions. A missing or renamed
  Composition makes that surface's admission binding deny requests.
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
- A **Grafana Cloud organization access policy token** for every registered organization, with only
  the organization-level capabilities needed to manage stacks and stored in your secret manager (see
  [Secrets](secrets.md)).
- A Grafana Cloud **region slug** for the stack you intend to create (e.g.
  `prod-us-central-0`).

## Pinned versions

Use the [installation version table and signature requirements](installation.md#status-and-pinned-versions). The admission gate is implemented, but the 1.0 candidate remains held by owner decision; see the [release boundary](migration-1.0.md).

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
- `spec.organization` — an immutable registered organization key. Its registry entry must permit
  both this region and the selected usage.
- `spec.usage` — an immutable platform-approved classification (`development` or `production` in
  the reference vocabulary) that becomes part of the output path
  `{outputSecretPrefix}/{organization}/{usage}/{slug}`. It cannot be changed after creation because
  it is part of external credential identity.
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

See [Request Schema Reference](reference/request-schema.md) for what every field does.

The example omits `spec.lifecycle.externalResources`, so it uses the safe `Retain` default. An
authorized `Delete` value is a decommission intent only: it requires an exact platform-owned
authorization for the request namespace, name, Kubernetes UID, and immutable profile,
the first reviewed request change, and a wait for `status.deletionReady=true`. Stage 2 removes
dependent access claims and waits for their Kubernetes objects and finalizers to be gone while the
Stack still exists; Stage 3 then removes the request.

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
## Direct-apply command reference

Copy the minimal catalog directory into `enabled/`, edit every placeholder, select a real Grafana Cloud region, and review every optional feature. Apply it directly for an evaluation:

~~~bash
cp -R examples/catalog/minimal enabled/my-stack
kubectl apply -k enabled/my-stack
~~~

For GitOps, commit the enabled directory and let the ApplicationSet create one Argo CD Application for it.

### 6. Observe reconciliation

~~~bash
kubectl get grafanacloudstackrequests -n grafana-vending
kubectl describe grafanacloudstackrequest -n grafana-vending REPLACE_WITH_SLUG
kubectl get managed -n grafana-vending
kubectl get pushsecrets,externalsecrets -n grafana-vending
kubectl get providerconfigs.grafana.m.crossplane.io -n grafana-vending
~~~

The request becomes Ready only when all currently desired composed resources report Ready. Rotating-token resources are intentionally rendered one reconciliation after Grafana assigns the parent service-account or policy ID.


## Next steps

- [Installation](installation.md) — bootstrapping Crossplane, ESO, and the platform components.
- [Request Schema Reference](reference/request-schema.md) — the complete request API field reference.
- [Configuration](configuration.md) — platform policy, profiles, and the organization registry.
- [Secrets](secrets.md) — how per-organization credentials get in, and how per-stack tokens get
  out.
- [Architecture](architecture.md) — the three-controller split and reconciliation model.
