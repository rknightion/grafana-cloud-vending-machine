# Cross-stack project content

## Credential ownership and lifecycle decision

The `GrafanaProjectContent` composite owns its central-stack access policy,
bounded-lifetime token and credential publication. It does not own either stack.
The platform administrator owns the profile that authorizes both targets, the
project label equality, backend connection details and Git repository. The claim
cannot supply a broader selector, a provider configuration or a credential value.

The credential belongs to project content, independently of the project stack's
lifetime. It uses the existing StackConsumer credential machinery and platform
maximum lifetime. This feature adds no rotation mechanism or interval change.
Deleting a claim retains external resources, following the existing consumer
contract; deleting a project stack does not revoke its central credential.

Revocation is an explicit platform operation: pause reconciliation of the owned
access policy and rotating token before revoking them in the central stack, then
remove the published credential and its project-side materialization. Revoking
only a token while its rotating-token controller remains active is not durable
revocation. Deleting only a claim is not revocation either. A central revocation
causes queries through the retained project datasources to fail authentication;
there is no fallback to a broader credential. Crossplane readiness alone does
not prove that backend queries succeed or detect an out-of-band revocation.

Credential publication retains `PushSecret.refreshInterval: 1h` and
`updatePolicy: Replace`. Plan for hour-scale propagation, not immediate delivery;
the downstream ExternalSecret refresh adds another reconciliation delay. These
intervals are not a guaranteed minimum latency or maximum outage bound. A revoked
credential can remain in the project-side secret until replacement propagates,
but keeping its bytes does not restore server-side authorization.

## Bounded contract and trigger

One approved profile selects one existing project stack and one existing central
stack. The fixed set is two observe-only Stack resources, one central access
policy and token, one PushSecret, one datasource-credential ExternalSecret, two
datasources (metrics and logs), one folder, and the existing Git Sync credential,
secure value, connection and repository resources. Synced dashboards are owned
by Git Sync, not individually adopted by this claim.

The external project trigger, approval boundary and field mapping are defined in
[External project integration](external-project-integration.md). That shared
contract also governs this content-only claim.

Provisioning is staged because provider-assigned identities and credentials must
be observed first. It is not a transaction across two Grafana stacks. An invalid
profile produces no new content; a provider refusal leaves the composite unready
and records which stack refused. The renderer reconstructs the complete authored
set from the approved profile, credentials and observed provider identities on
every reconcile where those dependencies are available. Provider refusal changes
status, not desired membership. Observed spec drift is not copied into desired
configuration. An unavailable stack ID can be recovered from its existing policy
realm, and an unavailable policy ID from its existing rotating token. Backend
connection details must still come from the central observer; an existing
datasource's URL or tenant configuration is not an authorized recovery source.

If a required identity or credential cannot be recovered, an existing request is
unrenderable and returns Fatal. In the pinned Crossplane 2.3.4 core,
[`composition_functions.go`](https://github.com/crossplane/crossplane/blob/c2670a735103660d7f436585447f2071bc1de973/internal/controller/apiextensions/composite/composition_functions.go#L460-L461)
returns from composition at the Fatal branch before garbage collection (line 570)
and composed-resource apply (line 628). This preserves existing children but does
not apply new custom status. Initial invalid requests have no children to
withdraw and return side-specific refusal status and one warning without Fatal.

## Claim and profile

An inert claim shape is:

```yaml
apiVersion: platform.example.org/v1beta1
kind: GrafanaProjectContent
metadata:
  name: example-project
  namespace: example-projects
spec:
  profile: example-project
```

The shipped Composition authorizes no projects. Its administrator supplies
`projectContentProfiles` in the function input. Each entry has `name`, `namespace`,
`consumerProfile`, `projectStack` (`slug`, `region`, `providerConfigName`),
`centralStack` (`slug`, `region`), `projectLabel` (`name`, `value`), `git`, and
`repository`. Both stacks must be observable through the selected consumer's
organization provider. The project instance ProviderConfig must already exist
and be bound by the administrator to the declared project stack.

`consumerProfile` selects a complete `stackConsumerProfiles` entry in this
Composition input: authorized namespace and central target, organization
ProviderConfig, unique consumer name and output path, and exactly `metrics:read`
and `logs:read`. Reserve that consumer identity exclusively for this project;
do not configure a separate StackConsumer claim to own the same resources.
Sharing a consumer profile across project profiles is refused. Project labels
are compiled as one escaped equality selector on the central access-policy realm.
Backend URLs and tenant IDs are read from the observed central Stack status.

`git` uses the existing GrafanaProvisioningConnection spec fields except
`stackRef`, which the renderer supplies. It requires the GitHub App metadata,
credential remote reference, one decrypter, secure version, title, description,
type and URL. `repository` uses the existing typed GrafanaProvisioningRepository
repository fields with `sync.target: folder`; the renderer supplies the UID and
connection reference. The repository's Git Sync root is managed by Git Sync;
the separately rendered project folder is available for manually managed content
and is not asserted to be the repository root.

Generated resource names use the profile name plus fixed suffixes. Provider
resources with derivable import identities carry `crossplane.io/external-name`.
The claim profile is immutable. Changes to a profile's targets, credential
identity or label restriction require an explicit platform migration rather than
reusing this claim to transfer ownership.

`status.projectContent.projectStack` and `.centralStack` distinguish `Waiting`,
`Provisioning`, `Refused`, and `Ready`. `.ready` means all expected children have
reported readiness. A `Synced=False` child produces a refusal for its owning side;
the composite does not copy potentially sensitive provider error text. Inspect
the named managed resources for detail. Readiness is reconciliation evidence,
not a successful query or dashboard-sync probe.
