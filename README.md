# Grafana Cloud vending machine with Crossplane

This repository is a portable reference architecture for vending Grafana Cloud stacks through a small, declarative API. Argo CD owns what is in Git, Crossplane continuously reconciles Grafana Cloud, and External Secrets Operator (ESO) moves credentials between Kubernetes and an external secret store.

It is deliberately more than a minimal stack example. The baseline includes rotating administrator and telemetry credentials, deletion protection, stack-local provider configuration, starter content, configurable drift behavior, OAuth and SAML SSO, reports, plugins, incident relay resources, teams, basic-role ACLs, fixed-role assignments, custom roles, and role assignments.

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
| Vending composition function | sha256:36aee084bda51e1bd909ee40ce8f2144808130472b30d5f77f8e41162d0e2e2f | Signed amd64/arm64 package built from commit 7c3a37bdd482 |

The Grafana Crossplane provider describes itself as experimental and unsupported. Version 2.13.0 is the latest Crossplane-provider release as of 2026-08-04, but it was generated from Terraform provider 4.40.0 while Terraform provider 4.43.0 is current. In particular, Terraform provider 4.40.1 contains a Stack drift fix that is not in this Crossplane provider release. Test provider upgrades and drift behavior against non-production stacks before rollout.

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
│   ├── catalog/              inert stack, SSO, Teams, RBAC, and content-ACL examples
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
| spec.incidentIntegration | Optional relay-backed OnCall and Alerting endpoints plus OnCall template ownership mode |

Three access APIs sit beside the stack request:

| API | Ownership unit | Use it for |
| --- | --- | --- |
| GrafanaCustomRoleBinding | One Team, one custom Role, one whole-role RoleAssignment | Compact compatibility pattern for a unique custom role |
| GrafanaTeamAccess | One Team plus zero-to-many custom and fixed-role assignments | Directory sync, direct membership, team preferences, additive custom roles, and existing fixed roles |
| GrafanaContentAccessPolicy | The complete ACL for one Folder or Dashboard | Basic-role, team, or user grants with one unambiguous owner per target |

GrafanaTeamAccess uses RoleAssignmentItem rather than the whole-set RoleAssignment resource. That makes independently owned team bundles additive. GrafanaContentAccessPolicy deliberately uses the whole-set FolderPermission or DashboardPermission resource; only one policy may own a given target.

The XRD uses defaultCompositionUpdatePolicy: Automatic and an enforced Composition reference. Existing requests therefore move to the latest Composition revision automatically after a platform update. Treat an XRD or function change like a production API release: render it, inspect the desired-resource diff, and roll it through a non-production request first.

## Platform configuration

The Composition input in platform/apis/v1beta1.yaml is the platform-owned policy boundary. It controls:

- organizationProviderConfigName: the namespaced ProviderConfig used for organization-level Cloud operations;
- outputSecretPrefix: the external path prefix for generated per-stack documents;
- secretStoreRef: either a namespaced SecretStore or a ClusterSecretStore;
- ssoProfiles: approved OAuth or SAML settings and Secret references;
- incidentProfiles: approved relay URLs and authorization Secret references.

Consumers select a profile by name. They cannot supply an arbitrary identity endpoint, client secret, incident URL, or authorization value in a stack request.

The API group is not a runtime setting. Replace every occurrence of platform.example.org in the XRDs, Compositions, examples, tests, Argo health configuration, and documentation when making a production fork.

## Catalog and adaptation map

Nothing in examples/catalog is live. Each directory demonstrates one ownership decision and uses reserved example values.

| Directory | What it demonstrates | What an adopter changes |
| --- | --- | --- |
| minimal | Safe stack baseline, rotating credentials, create-only content, no SSO | Slug, region, usage, API group, secret backend |
| comprehensive | Enforced content and OAuth SSO, report, plugin, incident relay, compact custom role | All profile names, endpoints, recipients, plugins, role actions/scopes |
| sso-create-only | Platform initializes OAuth, then stack administrators own later SSO edits | Approved OAuth profile and handoff policy |
| sso-azuread | Azure AD OAuth profile selected from platform policy | Tenant/application IDs, group claims, role expression, client-secret path |
| sso-saml | SAML metadata and role-value mapping | Metadata URL, attributes, signing requirements, role values |
| access-and-rbac | Direct and directory Team membership, preferences, custom/fixed roles, folder/dashboard ACLs | Team/group names, verified role UIDs, actions/scopes, ACL targets |

Copy a directory as a starting point; do not enable every option merely because it exists. A production catalog normally offers a few reviewed profiles such as standard, regulated, and administrator-owned SSO rather than exposing raw provider fields to request authors.

## SSO patterns and ownership

The provider supports OAuth settings for github, gitlab, google, azuread, okta, and generic_oauth, plus SAML. This reference accepts those Grafana Cloud-relevant providers. The upstream schema also contains LDAP settings, but LDAP is a self-managed Grafana integration rather than a portable Grafana Cloud stack profile, so the function rejects it.

Platform owners define complete profiles in the Composition input. Request authors can select only a profile name and ownership mode. OAuth client secrets must be LocalSecretKeySelector references; literal client-secret fields are not copied by the function. SAML can use an IdP metadata URL, or certificateSecretRef and privateKeySecretRef when the chosen flow requires key material.

| Pattern | Important profile fields | Role behavior |
| --- | --- | --- |
| Generic OAuth/OIDC | authUrl, tokenUrl, apiUrl, clientId, scopes, claim paths | roleAttributePath maps claims to Viewer, Editor, Admin, or None |
| Azure AD | tenant-specific auth/token URLs, clientId, group claim path | group-aware roleAttributePath; account for group overage behavior in the IdP design |
| GitHub, GitLab, Google, or Okta | provider-specific organizations/domains/groups and client credentials | Use allowed groups/organizations as an admission gate, then map the resulting role |
| SAML | idpMetadataUrl, assertion attributes, roleValues fields, signature settings | IdP role values map to Grafana basic roles |

roleAttributeStrict=true rejects a login when no valid role can be derived. skipOrgRoleSync=true has the opposite ownership implication: Grafana stops updating the user's organization basic role from the IdP. Use it only when another reviewed process owns organization roles. allowAssignGrafanaAdmin is far more privileged than organization Admin and should remain false unless server-administrator assignment is explicitly required and supported.

The four reconciliation modes apply to every supported profile type:

- enforced keeps the selected settings in forProvider and repairs UI drift;
- createOnly puts the settings in initProvider and preserves later administrator changes;
- observeOnly supplies only providerName with Observe permission;
- disabled emits no SSO managed resource.

Keep providerName stable when switching ownership. A change from generic_oauth to saml is an identity-provider migration, not a routine mode toggle, and needs a tested login and rollback plan.

## Teams, basic roles, fixed roles, custom roles, and content ACLs

Grafana authorization has several layers that should not be collapsed into one giant role document:

1. SSO maps a user to the organization basic role Viewer, Editor, Admin, or None.
2. Teams group users through direct membership and/or identity-provider Team Sync.
3. Grafana-managed fixed roles provide reusable capability bundles and can be assigned to a Team by UID.
4. Custom roles contain explicit action/scope pairs and are assigned to Teams.
5. Folder and dashboard ACLs grant View, Edit, or Admin to a basic role, Team, or individual actor for one content target.

The current provider does not offer a resource that rewrites the global definitions of the built-in Viewer, Editor, and Admin basic roles. Do not model that as if it did. The supported customization points are SSO basic-role mapping, fixed/custom role assignments, and per-folder or per-dashboard permissions. Fixed roles are Grafana-managed definitions. Assignment uses the role UUID, not its display name: for example, the reviewed catalog maps fixed:datasources:reader to fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ. Inventory the target stack before using any UUID because availability varies by version, edition, entitlement, and stack creation date.

GrafanaTeamAccess supports both members and externalGroups. Direct members must already exist in Grafana. External groups require the applicable Team Sync capability. With ignoreExternallySyncedMembers=true, provider membership reconciliation ignores members supplied by Team Sync while still managing the direct member set declared in Git. Give each Team one Kubernetes owner.

Custom role permissions are an allow list; there is no deny rule. Prefer the narrowest action and scope, keep global=false for stack-local roles, and validate actions against the Grafana version deployed to the target stack. The access-and-rbac example demonstrates alert-rule read/create/write plus the supporting folder-read and data-source-query permissions rather than granting organization Admin. Narrow folders:* and datasources:* to named UIDs in a real catalog.

RoleAssignment manages the entire set of actors for a role and conflicts with RoleAssignmentItem. GrafanaCustomRoleBinding is safe only because it creates a unique role and owns that role's entire assignment set. GrafanaTeamAccess uses RoleAssignmentItem so multiple team bundles can add assignments independently. Never manage the same role/actor pair through both APIs.

FolderPermission and DashboardPermission also manage the entire ACL. Omitting an entry removes it on the next enforced reconciliation. Put all grants for a target into one GrafanaContentAccessPolicy, including basic-role grants that should remain. The reference omits Delete permission from the managed resource lifecycle, so deleting the Kubernetes policy orphans the last external ACL instead of clearing it; edits while the policy exists are still repaired.

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

The optional profile secrets are intentionally excluded from deploy/aws/kustomization.yaml. Apply deploy/aws/optional-profile-secrets.yaml only after the corresponding remote secrets and profile definitions are ready. The file includes OAuth inputs for example-oidc and example-azuread plus the incident relay input; the example-saml profile uses public IdP metadata and needs no committed key material.

### 5. Enable one request

Copy one catalog directory into examples/enabled, edit the slug in both places, select a real Grafana Cloud region, and review every optional feature. Apply it directly for an evaluation:

~~~bash
kubectl apply -f examples/enabled/my-stack/
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
| Team direct members and preferences | present in GrafanaTeamAccess | Crossplane restores the declared direct members and preferences; external sync members are ignored when configured |
| Fixed/custom RoleAssignmentItem | present in GrafanaTeamAccess | Crossplane restores the role-to-Team assignment without owning other actors assigned to that role |
| Folder or dashboard ACL | present in GrafanaContentAccessPolicy | Crossplane restores the complete declared ACL; omitted entries are removed by Grafana's whole-set permission API |
| OnCall outgoing-webhook data | createOnly | Initial generic payload is set; later UI template edits are preserved |
| OnCall outgoing-webhook data | enforced | Later UI template edits are restored from the Composition |
| Alerting contact-point payload | enabled | Crossplane always restores the relay payload and Secret-backed authorization contract |

Changing SSO from enforced to createOnly moves oauth2Settings or samlSettings from forProvider to initProvider while retaining a stable external name for the provider. This is the supported handoff from platform ownership to administrator ownership. Changing to observeOnly removes write authority. Changing to disabled removes the managed-resource object from the Composition; because Delete is not permitted, the external SSO configuration remains but is no longer observed.

Changing dashboards from enforced to createOnly stops ownership of configJson. It does not roll back whatever value existed at the handoff. Returning to enforced makes the Git-rendered dashboard authoritative again.

Argo CD should ignore Crossplane-generated resource churn rather than carrying broad ignore rules for the request itself. If the request in Git changes, Argo applies the request; the Composition then computes the managed-resource change.

## Baseline and optional resources

The baseline creates three portable starting points that occupy the same resource slots as a traditional billing/usage, endpoints, and home-dashboard bundle:

- Billing and usage
- Telemetry endpoints
- Stack home

They are intentionally simple and generic; source-specific dashboard JSON is neither copied nor implied. Replace the dashboard builder in platform/function/fn.go or create a separate per-stack content API for substantial dashboards. Keep service-owned dashboards outside the stack identity API so an ordinary content release cannot disturb stack identity or credentials.

The monthly report uses the billing dashboard, PDF and CSV formats, the previous calendar month, UTC, and a deterministic ten-minute schedule slot derived from the slug. It requires an applicable Grafana Cloud plan and report capability.

The incident option creates four OnCall outgoing webhooks—test/production firing and resolved—and two Alerting contact points. They call a platform-owned relay and contain only Secret references for authorization. The relay owns any destination credential and organization-specific payload translation. templateMode defaults to createOnly for OnCall data templates, matching the common handoff where stack administrators may tailor the event body; enforced makes the generic Git template authoritative. Contact-point payloads remain enforced.

Plugin installation is list-driven. Pin plugin versions for repeatability. latest is convenient for an evaluation but delegates upgrade timing to Grafana Cloud.

## Complete provider surface and ownership boundaries

Provider 2.13.0 exposes 111 namespaced external managed-resource kinds across 16 Grafana API families. Comprehensive architecture means assigning every family a sensible owner; it does not mean every new stack should automatically create an SLO, a k6 project, an incident schedule, an ML job, and organization members.

The activation policy enables only kinds emitted by the current Compositions. Add a kind deliberately when adding a domain API.

| Provider family | Treatment in this reference |
| --- | --- |
| alerting | ContactPoint is an optional stack child. Rule groups, notification policy, templates, mute timing, inhibition, recording rules, and enrichment belong in a separate per-stack alerting bundle with one owner for the routing tree; use the stack-local ProviderConfig and the same non-destructive policy pattern. |
| asserts | Use a separate opt-in onboarding module because entitlement and additional metrics/Grafana credentials are required. |
| assistant | MCP servers, quickstarts, rules, and skills require a separate security and content lifecycle. |
| cloud | Core owns Stack, StackServiceAccount, StackServiceAccountRotatingToken, AccessPolicy, AccessPolicyRotatingToken, and optional PluginInstallation. Organization membership, private data-source networking, and observability onboarding belong in separate approved modules. |
| cloudintegrations | CloudIntegration belongs in an integration module selected after stack creation. |
| cloudprovider | AWS scrape jobs/accounts and Azure credentials require separate cloud trust and approval. |
| connections | Metrics endpoint scrape jobs are workload-owned connection objects. |
| enterprise | Core optionally owns Report; access APIs own Role, RoleAssignment, and RoleAssignmentItem. Data-source policy, SCIM, Keeper, secure values, and standalone external-group mapping are separate security-sensitive modules. |
| fleetmanagement | Collectors and pipelines have an independent rollout lifecycle. |
| frontendobservability | Applications require workload identity and origin inputs unavailable at stack creation. |
| grafana | Namespaced ProviderConfig is created per stack. ClusterProviderConfig is avoided to preserve namespace isolation. |
| k6 | Projects, tests, load zones, limits, and schedules are independent domain objects. |
| ml | Alerts, holidays, jobs, and outlier detectors depend on real queries and service ownership. |
| oncall | Core optionally creates relay-backed OutgoingWebhook resources. Users, routes, schedules, shifts, integrations, and escalation policy belong in an incident-management module. |
| oss | Core owns Folder, Dashboard, OrganizationPreferences, and SsoSettings; access APIs own Team, FolderPermission, and DashboardPermission. Data sources, playlists, library panels, annotations, repositories, users, and additional service accounts belong in content or integration modules. |
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

This matrix was checked resource-by-resource against the active modules in the Terraform source used to design this reference. Commented-out examples and surrounding pipeline/cloud infrastructure are not counted as active stack behavior. Product-specific dashboard JSON, identity endpoints, external-system payloads, and secret paths are intentionally represented by neutral extension points rather than copied content.

| Terraform vending concern | Crossplane equivalent | Parity |
| --- | --- | --- |
| Cloud stack resource | Stack composed from GrafanaCloudStackRequest | Equivalent, continuously observed |
| Stack name, slug, region, usage, labels | Request fields and Stack labels | Equivalent |
| Stack readiness wait and first-create delay | waitForReadiness plus observed-ID gates and references | Equivalent outcome without a fixed sleep or two-pass apply |
| Deletion protection | Stack deleteProtection plus omitted Delete policy | Equivalent with a stronger default |
| Administrator service account | StackServiceAccount | Equivalent |
| Static administrator token | StackServiceAccountRotatingToken | Superset through automatic rotation |
| Stack-local provider | ESO-built credentials plus namespaced ProviderConfig | Equivalent without credentials in code or state |
| External credential document | PushSecret document containing name, slug, URL, region, usage, request references, token, and telemetry path | Field parity through a backend-neutral secret manager |
| Telemetry publisher | Stack-realm AccessPolicy, rotating token, separate output | Superset through least privilege |
| Plugins | PluginInstallation list | Equivalent provider support |
| Billing/usage, endpoints, and home folders/dashboards | Three Folder and Dashboard pairs | Resource and lifecycle parity; neutral starter JSON replaces source-specific content |
| Ignore dashboard JSON changes | initProvider configJson in createOnly mode | Equivalent |
| Enforce dashboard JSON | forProvider configJson in enforced mode | Additional explicit option |
| Organization home dashboard | OrganizationPreferences | Equivalent, create-only or enforced |
| Monthly billing/usage report | Report with prior-month range, PDF/CSV, recipients, reply-to, and spread schedule | Equivalent; deterministic scheduling replaces random state |
| Code-owned OAuth SSO | SsoSettings enforced mode | Equivalent; UI drift is repaired |
| Administrator-owned OAuth SSO | createOnly, observeOnly, or disabled | More explicit than blanket ignore rules |
| OAuth alternatives and SAML | Platform-owned profile catalog | Additional Azure AD and SAML examples; GitHub, GitLab, Google, Okta, and generic OAuth are supported |
| Directory-synchronized teams | GrafanaCustomRoleBinding to Team | Equivalent |
| Custom roles and permissions | GrafanaCustomRoleBinding to Role | Equivalent |
| Role assignment | GrafanaCustomRoleBinding to RoleAssignment | Equivalent |
| Direct members, team preferences, multiple custom roles | GrafanaTeamAccess | Additional reusable access pattern |
| Existing fixed-role assignment | GrafanaTeamAccess to RoleAssignmentItem | Additional least-privilege pattern |
| Basic-role and Team folder/dashboard ACLs | GrafanaContentAccessPolicy | Additional authoritative content-access pattern |
| Incident outgoing webhooks | Four optional OutgoingWebhook resources with create-only or enforced data templates | Structural and reconciliation parity through a generic relay |
| Alert contact points | Two optional ContactPoint resources | Structural parity through a generic relay |
| Random report scheduling state | Deterministic FNV-derived UTC slot | Stateless replacement |
| Initial-creation feature gate | Observed-resource dependency gates | Architectural replacement; no manual second phase |
| Per-stack directory vending | Argo CD ApplicationSet over enabled directories | Equivalent GitOps request boundary |
| Plan/apply pipeline | Argo self-heal plus Crossplane reconciliation | Continuously reconciled replacement |
| Per-stack remote state and lock | Kubernetes API and Crossplane state | Architectural replacement |
| Destructive branch deletion | Prune Kubernetes objects and orphan externals | Intentional non-parity; destruction is separately approved |

### Capability completeness versus automatic baseline

“Covered” does not always mean “created for every stack.” The stack identity API automatically creates only resources that are safe and meaningful without workload context. The following high-value provider capabilities were outside the Terraform source and remain opt-in domains:

| Use case | Reference position | Why it is not automatic |
| --- | --- | --- |
| Alert rules, notification policy, mute timings, templates | Separate per-stack alerting bundle | The notification-policy tree is a whole-set singleton and needs one owner |
| Data sources and data-source permissions | Separate connection/access bundle with Secret references | Endpoints, credentials, and network trust are workload-specific |
| Private data-source connect | Separate approved network module | Creates network trust and tokens outside ordinary stack vending |
| Cloud integrations and scrape jobs | Separate cloud-integration module | Requires cloud-account permissions and approval |
| Additional service accounts and service-account permissions | Separate automation identity bundle | Role, token audience, owner, and rotation policy differ per workload |
| SLOs and Synthetic Monitoring | Service-owned definitions using the stack ProviderConfig | Objectives, queries, probes, and targets cannot be inferred safely |
| OnCall schedules, escalation chains, routes, and integrations | Incident-management bundle | People, rotations, and escalation policy have an independent lifecycle |
| Frontend Observability, k6, Fleet Management, ML, Asserts, Assistant | Dedicated domain modules | Each has entitlement, identity, content, and rollout inputs beyond stack creation |

The complete provider-family table above is the extension index. New modules should reuse the namespaced stack ProviderConfig, keep secrets in external stores, choose whole-set versus item resources deliberately, omit Delete by default, and document their reconciliation owner in this README.

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

The unit tests pin the desired-resource contracts, gating behavior for observed IDs, rotating-token parameters, least-privilege scopes, output-document shape, reconciliation modes, OAuth and SAML rendering, SSO Secret references, incident resources, Team membership/preferences, custom and fixed-role assignments, whole-target content ACLs, and safe composite status.

Before making the repository public, also review the complete Git history and repository settings. The working-tree scan cannot prove that an earlier private commit never contained a secret.

## Known limitations

- The Grafana provider is experimental and may lag the Terraform provider.
- Provider schemas and Grafana APIs may expose fields that do not round-trip cleanly; test drift rather than assuming.
- The reference has no destructive workflow.
- It does not provision Kubernetes, AWS infrastructure, DNS, identity providers, or incident relays.
- It uses generic starter dashboards rather than a full observability content library.
- Report, Enterprise, OnCall, plugin, and other resources require the relevant Grafana Cloud capabilities.
- SSO profiles are examples and must be replaced with reviewed identity settings.
- Built-in Viewer, Editor, and Admin definitions cannot be globally rewritten through the current provider; use SSO mapping, fixed/custom roles, and content ACLs.
- Fixed role UIDs and available RBAC actions vary by Grafana version, edition, and entitlement; inventory and test them before assignment.
- FolderPermission, DashboardPermission, RoleAssignment, NotificationPolicy, and similar whole-set APIs need exactly one declarative owner per external target.
- Crossplane readiness reports only the child resources currently desired; an optional disabled domain is not health-checked.
- Secret-store publication is eventually consistent with the configured ESO refresh interval.
- A successful render or unit test does not prove acceptance by a specific Grafana Cloud region or account. Use a disposable stack for live acceptance.

## License

Apache License 2.0. See LICENSE.
