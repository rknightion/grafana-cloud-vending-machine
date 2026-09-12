---
title: Installation
description: Bootstrapping Crossplane, External Secrets Operator, and the platform components, either directly or through Argo CD
---

# Installation

The 1.0 candidate remains held by owner decision. The admission repairs and real API-server gate are implemented; the [migration guide](migration-1.0.md) distinguishes the repaired defects from the continuing release hold. These instructions describe the unreleased reference.

This page covers bootstrapping the platform itself. Once it is installed and healthy, vending a
stack is the copy-edit-review-commit path in [Getting started](getting-started.md).

Review every manifest and replace `platform.example.org`, the region, repository URLs, secret
paths, profiles, and the function package reference before treating either path below as
production. See [Configuration](configuration.md) for platform policy and profiles, and [Request Schema Reference](reference/request-schema.md)
for the per-API fields.
## Status and pinned versions

This reference pins versions and immutable artifacts instead of following latest tags.

The cluster must run **Kubernetes 1.30 or later**. The `platform/` base installs
`admissionregistration.k8s.io/v1` `ValidatingAdmissionPolicy` resources, which are available as a
stable API from Kubernetes 1.30. On an older cluster, applying or syncing the platform base fails
before requests can be admitted.

| Component | Version | Source | Why |
| --- | --- | --- | --- |
| Kubernetes | 1.30+ | `platform/kustomization.yaml` (`apis/cloud-integrations-v1beta1.yaml`) | Minimum version for the `admissionregistration.k8s.io/v1` `ValidatingAdmissionPolicy` resources installed by the platform base |
| Crossplane | 2.3.4 | `deploy/argocd/crossplane.yaml` (`targetRevision: 2.3.4`) | Required for namespaced composite resources, namespaced managed resources, and ManagedResourceActivationPolicy |
| Grafana Crossplane provider | v2.14.0, immutable digest | `platform/provider/provider-grafana.yaml` (`refs/tags/v2.14.0`) | Tagged release generated from Grafana Terraform provider 4.45.1 with the complete upstream resource surface used here |
| ESO Helm chart | 2.6.0 | `deploy/argocd/external-secrets.yaml` (`targetRevision: 2.6.0`) | Last release before the open AWS PushSecret creation regression in 2.7.0 and 2.8.0 |
| Cosign verification image | 3.1.2, immutable digest | `platform/function/install.yaml`, `platform/provider/provider-grafana.yaml` (`cosign/cosign:v3.1.2`) | Verifies the Grafana provider and this repository's function package |
| Composition function SDK | 0.7.1 | `platform/function/go.mod` (`github.com/crossplane/function-sdk-go v0.7.1`) | Pinned by the function Go module |
| Vending composition function | sha256:50168c25dc02e98da3c16603919959f4a025f00aca685baa58e9616c566a29a8 | `platform/function/install.yaml` (`function-grafana-vending@sha256:50168c25dc02e98da3c16603919959f4a025f00aca685baa58e9616c566a29a8`) | Signed amd64/arm64 package |

The Grafana Crossplane provider describes itself as experimental and unsupported. The v2.14.0 tag is
generated from Terraform provider 4.45.1 and carries the resource surface used by this reference. It
is pinned by digest and Cosign-verified against the provider's `ci_tag.yaml` identity scoped to
`refs/tags/v2.14.0`. Test provider upgrades and drift behavior against non-production stacks before
rollout.

ESO issue [external-secrets/external-secrets#6593](https://github.com/external-secrets/external-secrets/issues/6593) remains open. Versions 2.7.0 and 2.8.0 send an empty replica-region request when creating an AWS Secrets Manager PushSecret target, which AWS rejects. Do not add a replica region merely to hide the bug. Upgrade after a fixed release exists and prove creation of a brand-new remote secret before removing the pin.

## Supply-chain controls

The Grafana provider manifest:

- pins an immutable OCI digest, carried identically in spec.package and in the verification job's argv;
- verifies Grafana's keyless signature against the exact publishing workflow identity scoped to `refs/tags/v2.14.0`;
- runs the provider with SafeStart;
- activates only the managed-resource kinds used by this reference.

The repository function workflow:

1. tidies and checks the Go module;
2. runs race-enabled tests and vet;
3. builds amd64 and arm64 distroless images from pinned bases;
4. assembles a multi-platform Crossplane package;
5. publishes an immutable commit-derived version;
6. signs the OCI index with keyless Cosign.

platform/function/install.yaml must pin the resulting signed digest for production. A fork must also change the package repository and the expected Cosign workflow identity. If the package is private, provide a dedicated read-only registry credential through an external secret; do not commit a Docker config or reuse a developer token.

The supplied install manifest verifies the pinned function package against this repository's exact main-branch workflow identity before Crossplane installs it. The verification Job name contains the digest prefix, so changing the digest creates a new gate rather than reusing an old successful Job.


## Releases

Pushes to `main` run release-please. Use Conventional Commits: `feat:` creates a minor release;
`fix:` and `perf:` create patch releases; a `!` marker or `BREAKING CHANGE:` footer records a
breaking change. The manifest starts at `0.1.0`; a pre-1.0 breaking change advances to `1.0.0`.

Release automation mints a short-lived, repository-scoped broker token. It never needs a
long-lived personal token. Broker or OpenBao reachability and unseal state are infrastructure
prerequisites, not evidence that the source validation gate failed.

## Direct installation

Suitable for a disposable or evaluation cluster.

### 1. Prepare the repository

Fork or copy the repository, choose an API group under a domain you control, and replace
`platform.example.org` everywhere. Change the repository URLs and function package path to your
fork. Keep `enabled/` empty until the controllers, provider, secret store, and every registered
organization `ProviderConfig` is healthy in every namespace that will accept requests.

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

Configure workload identity before applying the `SecretStore`. Confirm every organization
credential exists at its configured remote path first — see [Secrets](secrets.md).

### 4. Install the platform and environment configuration

```bash
kubectl create namespace grafana-vending
kubectl apply -k platform
kubectl apply -k deploy/aws
```

`platform/` installs the Grafana provider (with signature verification), the vending
composition function (also signature-verified), the `ManagedResourceActivationPolicy`, the
XRDs and Compositions, and the extra composition RBAC needed for ESO's `PushSecret` and
`ExternalSecret` resources. `deploy/aws` is an example for one request namespace: a real overlay
needs the `SecretStore`, one organization `ExternalSecret`, and one same-named ProviderConfig per
registry entry in every namespace that will accept requests.

Six specialist XRDs - Asserts, cloud integrations, k6, ML, PDC, and service accounts - also install a
cluster-scoped `ValidatingAdmissionPolicy` and binding. Each binding uses its XRD file's named
Composition as the policy parameter. Install each XRD together with that Composition and keep the
Composition name unchanged. If the Composition is absent or renamed, the binding's
`parameterNotFoundAction: Deny` causes requests for that surface to be rejected at admission;
`kubectl` and Argo CD report the denial against the named policy.

Wait for the provider and function to become healthy:

```bash
kubectl wait provider.pkg.crossplane.io/provider-grafana \
  --for=condition=HealthyPackageRevision \
  --timeout=10m

kubectl wait function.pkg.crossplane.io/function-grafana-vending \
  --for=condition=HealthyPackageRevision \
  --timeout=10m

kubectl get providerconfig.grafana.m.crossplane.io -n grafana-vending
```

The optional profile secrets are intentionally excluded from `deploy/aws/kustomization.yaml`.
Apply `deploy/aws/optional-profile-secrets.yaml` only after the corresponding remote secrets and
profile definitions are ready — it contains OAuth inputs for the example `generic_oauth` and
`azuread` profiles plus the incident relay input. The example SAML profile uses public IdP
metadata and needs no committed key material.

### 5. Hand off to the first-request guide

The platform bootstrap is complete once the provider, function, `ProviderConfig`, and secret
handoffs are healthy. Follow [Getting started](getting-started.md) for the end-to-end
copy-edit-review-commit flow, including the first request and reconciliation checks.
### Installation command reference

These steps are suitable for a disposable or evaluation cluster. Review every manifest and replace the API group, region, repository, secret paths, profiles, and package reference before treating the result as production.

### 1. Prepare the repository

Fork or copy the repository, choose an API group under a domain you control, and update platform.example.org everywhere. Change the repository URLs and function package path to your fork.

Keep `enabled/` empty until the controllers, provider, secret store, and every registered
organization ProviderConfig are healthy.

### 2. Install Crossplane

~~~bash
helm upgrade --install crossplane crossplane \
  --repo https://charts.crossplane.io/stable \
  --version 2.3.4 \
  --namespace crossplane-system \
  --create-namespace \
  --values deploy/crossplane/values.yaml
~~~

Wait for the Crossplane and RBAC manager deployments to become Available.

### 3. Install ESO

~~~bash
helm upgrade --install external-secrets external-secrets \
  --repo https://charts.external-secrets.io \
  --version 2.6.0 \
  --namespace external-secrets \
  --create-namespace \
  --values deploy/external-secrets/values.yaml
~~~

Configure workload identity before applying the SecretStore. Confirm every organization credential
exists at its configured remote path.

### 4. Install the platform and environment configuration

~~~bash
kubectl create namespace grafana-vending
kubectl apply -k platform
kubectl apply -k deploy/aws
~~~

Wait for the provider and function:

~~~bash
kubectl wait provider.pkg.crossplane.io/provider-grafana \
  --for=condition=HealthyPackageRevision \
  --timeout=10m

kubectl wait function.pkg.crossplane.io/function-grafana-vending \
  --for=condition=HealthyPackageRevision \
  --timeout=10m

kubectl get providerconfig.grafana.m.crossplane.io -n grafana-vending
~~~

The optional profile secrets are intentionally excluded from deploy/aws/kustomization.yaml. Apply deploy/aws/optional-profile-secrets.yaml only after the corresponding remote secrets and profile definitions are ready. The file includes OAuth inputs for example-oidc and example-azuread plus the incident relay input; the example-saml profile uses public IdP metadata and needs no committed key material.



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

Argo CD pruning a `GrafanaCloudStackRequest` uses the default
`spec.lifecycle.externalResources: Retain`: it deletes the composite Kubernetes object and its
composed managed-resource objects, while external Grafana and secret-manager objects are
**orphaned rather than destroyed**. Stack-local content is not retained independently when an armed
decommission deletes the Stack; deleting the Stack destroys that content. An authorized `Delete`
value is a separate, three-stage decommission path: the
first reviewed request change arms intent and waits for `status.deletionReady=true` (observed
`deleteProtection=false`, deletion-managed rotating tokens, and finalized/currently synced
credential PushSecrets); Stage 2 removes dependent access claims and waits for their Kubernetes
objects and finalizers to be gone while the Stack still exists; Stage 3 removes the request. Armed
Delete removes only the Stack, administrator service account/token, telemetry
access policy/token, and administrator/telemetry `PushSecret` documents. See the
[decommission runbook](governance.md#decommission-runbook).

## Upgrade runbook

For a provider upgrade:

1. read the provider release and the underlying Terraform provider changelogs;
2. compare generated CRD schemas for every activated kind;
3. verify the OCI signature and pin the new digest;
4. run function unit/render tests against the new schemas;
5. deploy to a cluster with a disposable stack;
6. make controlled out-of-band changes for each reconciliation mode;
7. prove both rotating-token paths and creation of new PushSecret targets;
8. inspect the desired and observed state of every child before promotion.

For a function upgrade:

1. keep the XRD API backward compatible within v1beta1;
2. add tests for the new desired-resource contract;
3. publish and sign the multi-platform package;
4. verify the signature against the exact workflow identity;
5. pin the immutable digest in platform/function/install.yaml;
6. let Automatic Composition updates reconcile a disposable request first.

For a Crossplane upgrade:

1. read the release notes for changes to package revision naming, because a new revision id re-mints a revision for every installed package and exercises the runtime hand-off on all of them at once;
2. keep serviceAccountTemplate.metadata.name unset in both DeploymentRuntimeConfigs. Crossplane names the runtime ServiceAccount after the revision unless the runtime config supplies a name, and a supplied name makes it a single object shared by every revision. Package runtime objects are applied with server-side apply and ownerReferences is a merge-keyed list, so the incoming revision's controller reference merges alongside the outgoing one and the API server rejects the object with "Only one reference can have Controller set to true". crossplane/crossplane#7714 fixed this for the Service and the TLS secrets and left the ServiceAccount on the plain applicator;
3. after the upgrade, confirm both packages report Healthy=True and that a runtime pod exists for each. A package that is Installed=True Healthy=False with no pod fails the pipeline closed with DeadlineExceeded and no children to pick from, which stops every request from reconciling, deletions included.

## Next steps

- [Getting started](getting-started.md) — vend your first stack.
- [Secrets](secrets.md) — the per-organization credential and per-stack token flow in detail.
- [Architecture](architecture.md) — the ownership boundaries between Argo CD, Crossplane, and
  ESO.
