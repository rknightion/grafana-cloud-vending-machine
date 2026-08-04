# Grafana Cloud vending machine with Crossplane

This repository is a portable reference architecture for vending Grafana Cloud stacks through a small, declarative API. Argo CD owns what is in Git, Crossplane continuously reconciles Grafana Cloud, and External Secrets Operator (ESO) moves credentials between Kubernetes and an external secret store.

It is deliberately more than a minimal stack example. The baseline includes rotating administrator and telemetry credentials, deletion protection, stack-local provider configuration, starter content, configurable drift behavior, SSO, reports, plugins, incident relay resources, teams, custom roles, and role assignments.

Nothing under examples/catalog is applied by the supplied ApplicationSet. The examples/enabled directory starts empty. A user must copy, edit, review, and commit a request before it can create a stack.

> Important: platform.example.org is a documentation placeholder, not a production API group. Replace it with a domain your organization controls before adopting this API.

## Status and pinned versions

This reference pins versions and immutable artifacts instead of following latest tags.

| Component | Version | Why |
| --- | --- | --- |
| Crossplane | 2.3.4 | Required for namespaced composite resources, namespaced managed resources, and ManagedResourceActivationPolicy |
| Grafana Crossplane provider | 2.13.0, immutable digest | Current provider release when this reference was published; generated from Grafana Terraform provider 4.40.0 |
| ESO Helm chart | 2.6.0 | Last release before the open AWS PushSecret creation regression in 2.7.0 and 2.8.0 |
| Cosign verification image | 3.1.2, immutable digest | Verifies the Grafana provider and this repository's function package |
| Composition function SDK | 0.7.1 | Pinned by the function Go module |
| Vending composition function | sha256:2def3a7a00b2ecc445caa178837376c60889da3588596c5c72f9993fcf9e23a3 | Signed amd64/arm64 package built from commit c5b0453549db |

The Grafana Crossplane provider describes itself as experimental and unsupported. The currently published Terraform provider is newer than the version used to generate provider 2.13.0. In particular, Terraform provider 4.40.1 contains a Stack drift fix that is not in this Crossplane provider release. Test provider upgrades and drift behavior against non-production stacks before rollout.

ESO issue [external-secrets/external-secrets#6593](https://github.com/external-secrets/external-secrets/issues/6593) remains open. Versions 2.7.0 and 2.8.0 send an empty replica-region request when creating an AWS Secrets Manager PushSecret target, which AWS rejects. Do not add a replica region merely to hide the bug. Upgrade after a fixed release exists and prove creation of a brand-new remote secret before removing the pin.

## What is in the repository

~~~text
.
├── platform/
│   ├── apis/                 XRDs and pipeline Compositions
│   ├── function/             Go composition function, tests, package metadata
│   ├── provider/             Grafana provider, activation policy, signature gate
│   ├── rbac/                 minimum extra composition RBAC for ESO resources
│   └── kustomization.yaml
├── examples/
│   ├── catalog/              inert minimal, comprehensive, and SSO examples
│   └── enabled/              watched by the example ApplicationSet; empty by default
├── deploy/
│   ├── argocd/               controller, platform, and per-request GitOps examples
│   ├── aws/                  SecretStore, ExternalSecrets, ProviderConfig, IAM policy
│   ├── crossplane/           production-oriented Helm values
│   └── external-secrets/     production-oriented Helm values
├── scripts/                  validation and public-release safety scan
└── .github/workflows/        validation and signed multi-platform function publishing
~~~

The core product is platform/. Everything under deploy/ is an example integration layer and may be replaced with the equivalent tooling used by your organization.

## Architecture

~~~mermaid
flowchart LR
    Git[Git request] --> Argo[Argo CD]
    Argo --> XR[GrafanaCloudStackRequest]
    XR --> XP[Crossplane composition]
    XP --> Cloud[Grafana Cloud resources]
    AWSIn[AWS Secrets Manager\norganization and profile credentials] --> ESO[External Secrets Operator]
    ESO --> K8sIn[Kubernetes Secrets]
    K8sIn --> XP
    Cloud --> Generated[Kubernetes connection Secrets]
    Generated --> ESO
    ESO --> AWSOut[AWS Secrets Manager\nper-stack outputs]
~~~

There are four separate ownership layers:

1. Argo CD owns controller installation, platform definitions, environment configuration, and request objects stored in Git.
2. Crossplane owns the composed Kubernetes managed resources and continuously reconciles their external Grafana objects.
3. ESO owns materialization of secret inputs and publication of generated secret outputs.
4. Grafana Cloud remains the external system of record observed and corrected by the Grafana provider.

Argo CD must not also declare the managed resources emitted by the Composition. That would give two controllers ownership of the same Kubernetes objects.

## Vending API

The primary API is GrafanaCloudStackRequest at platform.example.org/v1beta1. It is namespaced so teams or environments can be separated with Kubernetes namespaces and RBAC.

| Field | Purpose |
| --- | --- |
| spec.displayName | Human-readable stack name |
| spec.slug | Immutable Grafana Cloud slug; must equal metadata.name |
| spec.region | Grafana Cloud region slug |
| spec.usage | Classification and output-secret path segment |
| spec.profile | Platform-defined policy/profile label |
| spec.changeReference | Optional request or change identifier published in output metadata |
| spec.configurationItemReference | Optional inventory identifier published in output metadata |
| spec.baselineDashboards.enabled | Enables three starter folders and dashboards |
| spec.telemetryAccess.enabled | Enables a least-privilege rotating publisher credential |
| spec.plugins | Optional plugin slug/version list |
| spec.reconciliation | Selects content ownership behavior |
| spec.sso | Selects SSO profile and reconciliation mode |
| spec.monthlyReport | Optional scheduled usage report |
| spec.incidentIntegration | Optional relay-backed OnCall and Alerting endpoints |

GrafanaCustomRoleBinding is a second namespaced API. One object creates one Team synchronized to identity-provider groups, one non-global custom Role, and one RoleAssignment that binds the role to the team. Keeping these bindings separate allows zero-to-many authorization bundles beside a stack request.

The XRD uses defaultCompositionUpdatePolicy: Automatic and an enforced Composition reference. Existing requests therefore move to the latest Composition revision automatically after a platform update. Treat an XRD or function change like a production API release: render it, inspect the desired-resource diff, and roll it through a non-production request first.

## Platform configuration

The Composition input in platform/apis/v1beta1.yaml is the platform-owned policy boundary. It controls:

- organizationProviderConfigName: the namespaced ProviderConfig used for organization-level Cloud operations;
- outputSecretPrefix: the external path prefix for generated per-stack documents;
- secretStoreRef: either a namespaced SecretStore or a ClusterSecretStore;
- ssoProfiles: approved OAuth settings and Secret references;
- incidentProfiles: approved relay URLs and authorization Secret references.

Consumers select a profile by name. They cannot supply an arbitrary identity endpoint, client secret, incident URL, or authorization value in a stack request.

The API group is not a runtime setting. Replace every occurrence of platform.example.org in the XRDs, Compositions, examples, tests, Argo health configuration, and documentation when making a production fork.

## Secret and token design

No Grafana credential belongs in Git, a request object, Composition input, status, or function log.

### Organization credential

Create a Grafana Cloud access policy token with only the organization-level capabilities needed to manage stacks and their Cloud resources. Store it in the external secret manager as JSON:

~~~json
{
  "cloud_access_policy_token": "REPLACE_SECURELY"
}
~~~

The example expects this document at /platform/grafana-cloud/organization/credentials. Do not put the real token directly in a shell command, terminal history, CI variable dump, or Kubernetes manifest. Use your secret-management workflow or the AWS CLI file input mechanism from a permission-restricted temporary file.

deploy/aws/secret-store-and-credentials.yaml materializes the value into a Kubernetes Secret formatted for the provider. The namespaced grafana-cloud-org ProviderConfig references that Secret. Namespace RBAC should prevent request authors from reading it.

### Rotating administrator token

For every stack, the Composition creates:

1. a StackServiceAccount with Admin role;
2. a StackServiceAccountRotatingToken after Grafana reports the service-account ID;
3. a Kubernetes connection Secret written by the rotating-token resource;
4. a PushSecret that exports a structured administrator document;
5. an ExternalSecret that reads the exported token and URL back into the stack namespace;
6. a stack-local ProviderConfig used for Grafana resources inside that stack.

The token lifetime is 30 days with a seven-day early rotation window. The PushSecret refresh interval is one hour, so a newly rotated token is copied to the external store well inside the overlap window.

The exported document has this shape:

~~~json
{
  "stack_name": "Example stack",
  "stack_slug": "example",
  "stack_url": "https://example.grafana.net",
  "stack_region": "prod-us-central-0",
  "usage": "development",
  "change_reference": "CHANGE-EXAMPLE",
  "configuration_item_reference": "CONFIG-EXAMPLE",
  "stack_service_account_token": "GENERATED",
  "telemetry_access_policy_secret_path": "/platform/grafana-cloud/stacks/REGION/USAGE/SLUG/telemetry-publisher"
}
~~~

The generated token comes from the connection Secret at reconciliation time; it is not embedded in rendered YAML.

### Rotating telemetry token

When telemetryAccess.enabled is true, the Composition creates a stack-realm AccessPolicy with only:

- stacks:read
- metrics:write
- logs:write
- traces:write

An AccessPolicyRotatingToken uses the same 30-day lifetime and seven-day early rotation window. A separate PushSecret publishes the token and policy metadata under the stack's telemetry-publisher path. Workloads should use this token for telemetry and never receive the administrator token.

Static StackServiceAccountToken, AccessPolicyToken, and ServiceAccountToken resources remain available in the upstream provider but are deliberately not used. Their rotating counterparts avoid creating a permanent credential lifecycle outside the control plane.

### AWS permissions

deploy/aws/iam-policy.json is the minimum policy shape for the example paths. Attach it to the external-secrets ServiceAccount through the workload-identity mechanism for your cluster:

- EKS Pod Identity: create a Pod Identity association for namespace external-secrets and ServiceAccount external-secrets.
- IRSA: annotate that ServiceAccount with its role and use the standard EKS OIDC trust relationship.
- Other Kubernetes platforms: use the cloud identity integration recommended for that platform.

Do not put long-lived AWS keys in SecretStore. The controller should obtain short-lived credentials from workload identity. Restrict CreateSecret, PutSecretValue, and TagResource to the output prefix; restrict read access to the input and output paths actually required.

The example uses AWS Secrets Manager, not Systems Manager Parameter Store. The function itself only emits SecretStore references, so another ESO provider can be substituted if it supports ExternalSecret and PushSecret with the required structured-value behavior.

PushSecret uses deletionPolicy: None. Removing a request does not delete its external credential documents.

## Supply-chain controls

The Grafana provider manifest:

- pins the v2.13.0 OCI digest;
- verifies Grafana's keyless signature against the exact tag workflow identity;
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

## Follow along: direct installation

These steps are suitable for a disposable or evaluation cluster. Review every manifest and replace the API group, region, repository, secret paths, profiles, and package reference before treating the result as production.

### 1. Prepare the repository

Fork or copy the repository, choose an API group under a domain you control, and update platform.example.org everywhere. Change the repository URLs and function package path to your fork.

Keep examples/enabled empty until the controllers, provider, secret store, and organization ProviderConfig are healthy.

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

Configure workload identity before applying the SecretStore. Confirm that the organization credential exists at the configured remote path.

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

kubectl get providerconfig.grafana.m.crossplane.io \
  -n grafana-vending grafana-cloud-org
~~~

The optional profile secrets are intentionally excluded from deploy/aws/kustomization.yaml. Add deploy/aws/optional-profile-secrets.yaml only after the corresponding remote secrets and profile definitions are ready.

### 5. Enable one request

Copy one catalog directory into examples/enabled, edit the slug in both places, select a real Grafana Cloud region, and review every optional feature. Apply it directly for an evaluation:

~~~bash
kubectl apply -f examples/enabled/my-stack/stack.yaml
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

## Follow along: Argo CD

deploy/argocd contains four building blocks:

1. crossplane.yaml installs Crossplane 2.3.4 with the reference values.
2. external-secrets.yaml installs the version-gated ESO release.
3. platform.yaml installs platform/ plus the environment-specific AWS SecretStore and ProviderConfig.
4. requests-applicationset.yaml creates one Application for each directory under examples/enabled.

The examples assume an Argo CD AppProject named platform and a repository that Argo CD can read. While a fork is private, configure repository access through Argo CD's credential mechanism rather than placing credentials in an Application.

Apply the relevant settings from deploy/argocd/argocd-values.yaml to Argo CD:

- application.resourceTrackingMethod: annotation prevents Crossplane-generated children from inheriting Argo ownership by label;
- the health customizations understand Crossplane package, managed-resource, and composite conditions;
- ProviderConfigUsage is excluded because it is an internal Crossplane bookkeeping object.

The Application sync waves order controllers before platform APIs and requests. SkipDryRunOnMissingResource is needed while CRDs are appearing. Server-side apply avoids client-side annotation-size limits from large CRDs.

### GitOps ownership and pruning

Argo CD selfHeal repairs changes to request objects and platform manifests. Crossplane repairs changes to external Grafana resources according to management policies. These are different reconciliation loops.

Argo pruning a GrafanaCloudStackRequest deletes the composite Kubernetes object and its composed managed-resource objects. The managed resources omit the Delete management policy, the Stack enables delete protection, rotating tokens set deleteOnDestroy to false, and PushSecret uses deletionPolicy: None. External Grafana and secret-manager objects are therefore orphaned rather than destroyed.

That safety contract is intentional. A destructive decommission must be a separately reviewed operation with an inventory, dependency order, credential revocation, explicit deletion policies, and confirmation of the exact stack identity.

## Reconciliation and out-of-band changes

The short answer is: an administrator's SSO edit is automatically repaired only when SSO mode is enforced.

Crossplane providers poll the external APIs and compare observed state with desired state. The exact delay is the provider poll interval plus API and controller latency; it is not an immediate webhook response.

| Resource or field | Mode | Effect of an out-of-band edit |
| --- | --- | --- |
| Stack name, region, labels, readiness flags | managed forProvider fields | Crossplane attempts to restore the value in the request |
| Stack deletion | deleteProtection plus no Delete policy | The reference refuses normal controller-driven deletion |
| Baseline dashboard JSON | createOnly | Initial JSON is supplied through initProvider; later UI edits are preserved |
| Baseline dashboard JSON | enforced | UI edits are detected and restored from Git-rendered desired state |
| Folder title and dashboard folder relationship | managed forProvider fields | Drift is restored even when dashboard JSON is create-only |
| Home dashboard UID | createOnly | Initial choice is set; a later administrator change is preserved |
| Home dashboard UID | enforced | A later administrator change is restored |
| SSO settings | enforced | Crossplane owns OAuth settings and repairs UI drift |
| SSO settings | createOnly | Crossplane initializes settings but does not own later changes |
| SSO settings | observeOnly | Crossplane observes the named provider and never creates or updates it |
| SSO settings | disabled | No SSO managed resource is desired |
| Plugins | listed | Crossplane reconciles listed installations; omitted plugins are not adopted |
| Custom roles, assignments, team sync | present in a binding | Crossplane repairs drift in the managed fields |

Changing SSO from enforced to createOnly moves oauth2Settings from forProvider to initProvider while retaining a stable external name for the provider. This is the supported handoff from platform ownership to administrator ownership. Changing to observeOnly removes write authority. Changing to disabled removes the managed-resource object from the Composition; because Delete is not permitted, the external SSO configuration remains but is no longer observed.

Changing dashboards from enforced to createOnly stops ownership of configJson. It does not roll back whatever value existed at the handoff. Returning to enforced makes the Git-rendered dashboard authoritative again.

Argo CD should ignore Crossplane-generated resource churn rather than carrying broad ignore rules for the request itself. If the request in Git changes, Argo applies the request; the Composition then computes the managed-resource change.

## Baseline and optional resources

The baseline creates three portable starting points:

- Billing and usage
- Telemetry endpoints
- Stack home

They are intentionally simple and generic. Replace the dashboard builder in platform/function/fn.go or create a separate per-stack content API for substantial dashboards. Keeping service dashboards outside the stack identity API avoids coupling ordinary content releases to stack provisioning.

The monthly report uses the billing dashboard, PDF and CSV formats, the previous calendar month, UTC, and a deterministic ten-minute schedule slot derived from the slug. It requires an applicable Grafana Cloud plan and report capability.

The incident option creates four OnCall outgoing webhooks—test/production firing and resolved—and two Alerting contact points. They call a platform-owned relay and contain only Secret references for authorization. The relay owns any destination credential and organization-specific payload translation.

Plugin installation is list-driven. Pin plugin versions for repeatability. latest is convenient for an evaluation but delegates upgrade timing to Grafana Cloud.

## Complete provider surface and ownership boundaries

Provider 2.13.0 exposes 111 namespaced external managed-resource kinds across 16 Grafana API families. Comprehensive architecture means assigning every family a sensible owner; it does not mean every new stack should automatically create an SLO, a k6 project, an incident schedule, an ML job, and organization members.

The activation policy enables only kinds emitted by the current Compositions. Add a kind deliberately when adding a domain API.

| Provider family | Treatment in this reference |
| --- | --- |
| alerting | ContactPoint is an optional stack child. Rule groups, notification policy, templates, mute timing, inhibition, recording rules, and enrichment belong in a separate per-stack alerting bundle with one owner for the routing tree. |
| asserts | Use a separate opt-in onboarding module because entitlement and additional metrics/Grafana credentials are required. |
| assistant | MCP servers, quickstarts, rules, and skills require a separate security and content lifecycle. |
| cloud | Core owns Stack, StackServiceAccount, StackServiceAccountRotatingToken, AccessPolicy, AccessPolicyRotatingToken, and optional PluginInstallation. Organization membership, private data-source networking, and observability onboarding belong in separate approved modules. |
| cloudintegrations | CloudIntegration belongs in an integration module selected after stack creation. |
| cloudprovider | AWS scrape jobs/accounts and Azure credentials require separate cloud trust and approval. |
| connections | Metrics endpoint scrape jobs are workload-owned connection objects. |
| enterprise | Core optionally owns Report; GrafanaCustomRoleBinding owns Role and RoleAssignment. Data-source policy, SCIM, Keeper, secure values, and external groups are separate security-sensitive modules. |
| fleetmanagement | Collectors and pipelines have an independent rollout lifecycle. |
| frontendobservability | Applications require workload identity and origin inputs unavailable at stack creation. |
| grafana | Namespaced ProviderConfig is created per stack. ClusterProviderConfig is avoided to preserve namespace isolation. |
| k6 | Projects, tests, load zones, limits, and schedules are independent domain objects. |
| ml | Alerts, holidays, jobs, and outlier detectors depend on real queries and service ownership. |
| oncall | Core optionally creates relay-backed OutgoingWebhook resources. Users, routes, schedules, shifts, integrations, and escalation policy belong in an incident-management module. |
| oss | Core owns Folder, Dashboard, OrganizationPreferences, SsoSettings, and Team. Data sources, permissions, playlists, library panels, annotations, repositories, users, and additional service accounts belong in content or access modules. |
| slo | SLO objectives and queries are service-owned, not inferred from a stack request. |
| sm | Synthetic Monitoring installation, probes, checks, and alerting require approved targets, execution locations, and a separate credential chain. |

This leads to a clean GitOps tree:

~~~text
platform Application
  providers, functions, activation policy, XRDs, Compositions, secret stores

stack-request Applications
  one GrafanaCloudStackRequest
  zero or more GrafanaCustomRoleBinding objects

optional domain Applications
  stack content and permissions
  alerting
  cloud integrations and connections
  incident management
  Synthetic Monitoring and SLOs
  application, database, Kubernetes, and frontend observability
  Fleet Management, k6, ML, Asserts, and Assistant
~~~

## Terraform vending-machine equivalence

This is the practical mapping from a Grafana Cloud Terraform vending machine to the Crossplane reference.

| Terraform vending concern | Crossplane equivalent | Parity |
| --- | --- | --- |
| Cloud stack resource | Stack composed from GrafanaCloudStackRequest | Equivalent, continuously observed |
| Stack name, slug, region, usage, labels | Request fields and Stack labels | Equivalent |
| First-create delay | Observed ID gates and references | Safer replacement for a fixed sleep |
| Deletion protection | Stack deleteProtection plus omitted Delete policy | Equivalent with a stronger default |
| Administrator service account | StackServiceAccount | Equivalent |
| Static administrator token | StackServiceAccountRotatingToken | Superset through automatic rotation |
| Stack-local provider | ESO-built credentials plus namespaced ProviderConfig | Equivalent without credentials in code or state |
| External credential persistence | PushSecret to an external secret manager | Backend-neutral architectural replacement |
| Telemetry publisher | Stack-realm AccessPolicy, rotating token, separate output | Superset through least privilege |
| Plugins | PluginInstallation list | Equivalent provider support |
| Baseline folders and dashboards | Three Folder and Dashboard pairs | Resource parity with portable content |
| Ignore dashboard JSON changes | initProvider configJson in createOnly mode | Equivalent |
| Enforce dashboard JSON | forProvider configJson in enforced mode | Additional explicit option |
| Organization home dashboard | OrganizationPreferences | Equivalent, create-only or enforced |
| Monthly report | Deterministic optional Report | Equivalent resource shape |
| Code-owned OAuth SSO | SsoSettings enforced mode | Equivalent; UI drift is repaired |
| Administrator-owned OAuth SSO | createOnly, observeOnly, or disabled | More explicit than blanket ignore rules |
| Directory-synchronized teams | GrafanaCustomRoleBinding to Team | Equivalent |
| Custom roles and permissions | GrafanaCustomRoleBinding to Role | Equivalent |
| Role assignment | GrafanaCustomRoleBinding to RoleAssignment | Equivalent |
| Incident outgoing webhooks | Four optional OutgoingWebhook resources | Structural parity through a generic relay |
| Alert contact points | Two optional ContactPoint resources | Structural parity through a generic relay |
| Random report scheduling state | Deterministic FNV-derived UTC slot | Stateless replacement |
| Per-stack directory vending | Argo CD ApplicationSet over enabled directories | Equivalent GitOps request boundary |
| Plan/apply pipeline | Argo self-heal plus Crossplane reconciliation | Continuously reconciled replacement |
| Per-stack remote state and lock | Kubernetes API and Crossplane state | Architectural replacement |
| Destructive branch deletion | Prune Kubernetes objects and orphan externals | Intentional non-parity; destruction is separately approved |

## Migration and adoption

Do not point this platform at existing stacks casually. Adoption is a change of controller ownership, not just a manifest deployment.

A safe adoption rehearsal should:

1. record the stack slug, numeric ID, URL, region, deletion protection, SSO owner, service accounts, tokens, plugins, and current content;
2. back up the existing configuration through the supported Grafana APIs or its current IaC state;
3. render the request and inspect every desired managed resource before applying;
4. begin with create-only or observe-only modes where available;
5. verify that the Stack external name resolves to the intended existing identity;
6. confirm no unrelated service account, token, SSO provider, plugin, role, or dashboard would be claimed;
7. hand over one resource family at a time;
8. retain a tested rollback that removes Kubernetes ownership without deleting external resources.

The Stack resource uses the slug as crossplane.io/external-name, which provides an adoption path supported by the provider's external identity model. Other resources may require explicit external names or a dedicated migration Composition. Extend the request API rather than editing generated managed resources by hand.

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

## Decommission runbook

Normal Git deletion is non-destructive. After pruning, confirm that the external stack and credential documents remain and revoke any credentials that no longer have an owner.

If actual destruction is approved, use a separate change that:

1. resolves and records the exact stack ID and slug;
2. inventories dependants and data-retention requirements;
3. disables Argo auto-sync for the request while sequencing the operation;
4. revokes and removes publisher and administrator tokens;
5. deletes or reassigns stack-local resources in dependency order;
6. explicitly removes deletion protection and enables Delete only for the exact resources approved;
7. verifies external deletion and removes retained secret-manager outputs;
8. removes the request from Git after the external result is confirmed.

This repository intentionally contains no one-command destructive path.

## Validation

Run the complete local gate:

~~~bash
./scripts/validate.sh
~~~

It performs:

- public-release scanning for source identifiers, credential prefixes, private keys, local paths, account IDs, JWT-like values, Kubernetes Secret manifests, and sensitive file names;
- Go formatting and module consistency checks;
- race-enabled unit tests with coverage;
- go vet;
- YAML syntax parsing;
- Kustomize rendering for the platform and AWS examples.

The unit tests pin the desired-resource contracts, gating behavior for observed IDs, rotating-token parameters, least-privilege scopes, output-document shape, reconciliation modes, SSO Secret references, incident resources, custom roles, and safe composite status.

Before making the repository public, also review the complete Git history and repository settings. The working-tree scan cannot prove that an earlier private commit never contained a secret.

## Known limitations

- The Grafana provider is experimental and may lag the Terraform provider.
- Provider schemas and Grafana APIs may expose fields that do not round-trip cleanly; test drift rather than assuming.
- The reference has no destructive workflow.
- It does not provision Kubernetes, AWS infrastructure, DNS, identity providers, or incident relays.
- It uses generic starter dashboards rather than a full observability content library.
- Report, Enterprise, OnCall, plugin, and other resources require the relevant Grafana Cloud capabilities.
- SSO profiles are examples and must be replaced with reviewed identity settings.
- Crossplane readiness reports only the child resources currently desired; an optional disabled domain is not health-checked.
- Secret-store publication is eventually consistent with the configured ESO refresh interval.
- A successful render or unit test does not prove acceptance by a specific Grafana Cloud region or account. Use a disposable stack for live acceptance.

## License

Apache License 2.0. See LICENSE.
