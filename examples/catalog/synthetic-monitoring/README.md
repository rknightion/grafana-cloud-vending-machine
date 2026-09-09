# Synthetic Monitoring installation and bounded checks

This inert example installs Synthetic Monitoring for an existing stack and declares one HTTP check plus one browser check with provider-native alert criteria. Targets and probe locations are written by the consuming team because the vending platform cannot infer them from a stack request.

## Files

- `synthetic-monitoring.yaml` declares the installation and the complete check set owned by this composite.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Prerequisites

- The referenced stack is Ready and has telemetry access enabled. The installer reuses that stack's local, narrowly referenced telemetry token as bootstrap input; it does not mirror that input into the external secret store.
- The platform function can compose core Secrets, and External Secrets Operator can write to the configured secret store.
- The named probes exist and support the selected check type. Browser checks need browser-capable probes. The platform budget also names one approved public probe used only as input to a disabled verification check.
- Namespace RBAC denies consuming teams direct creation of provider `Check` resources. Otherwise direct or manual checks can bypass this composite's budget.

## Values to replace

- Replace `platform.example.org` and every `replacewithunique08` occurrence. Keep `metadata.name` equal to `spec.stackRef.name`; that identity makes one composite the owner of the stack's check set.
- Replace both example targets and probe names with reviewed service endpoints and approved execution locations. Review the per-check alert criteria and runbook URL with the service owner.
- Replace the browser script with the team's real check. Keep credentials outside the request and repository.

## Budget and ownership

The Composition selects a budget from the referenced stack's immutable usage class. It caps API and browser check counts separately, caps probe locations per check, sets a minimum interval, and rejects a check set whose hourly execution cost exceeds the profile. Browser executions carry a separate weight of ten in the reference profile. The `checks` list is the whole set owned by this composite; removing an item withdraws that managed check.

These limits govern checks created through this API. They are not a tenant service quota and cannot account for checks created manually or through another credential. Keep provider `Check` creation behind platform RBAC if the budget is a hard governance boundary.

## Alert criteria and private probes

Each check with an explicit nonempty `alerts` list owns one `CheckAlerts` resource after the provider reports that check's assigned ID. This relationship is deliberately deferred: the provider requires a numeric check ID, so the function never derives one from a name. `CheckAlerts` configures provider-native alert criteria, but its pinned schema has no contact-point, notification-policy, routing-tree, receiver, or label field. It therefore cannot prove delivery to a vended contact point; that connection requires an alerting policy owned outside this API.

Private probes are refused. The pinned provider's `Probe` resource has no token or token-lifetime field, so this API cannot satisfy the platform's bounded-credential requirement. It emits no `Probe`, probe token, status field, Secret, or catalog value.

## Readiness and credentials

The provider's Installation resource has a known upstream defect that can report Ready and Synced immediately even when the product was not configured. This composition ignores that condition for admission. It uses the derived Synthetic Monitoring token to create and observe a disabled HTTP check against a reserved `.invalid` target, then withholds team checks until that tenant-scoped API observation succeeds. The verification check never runs. Its configured public probe is an API input; public probe inventory is not accepted as installation proof.

The pinned provider exposes the derived token only through the Installation observation rather than as connection details. The function materializes that derived credential into a namespaced Secret and pushes only the derived token to the external secret store. The bootstrap telemetry token is referenced in place and is never copied by this module.

Existing checks that omit `alerts` retain their prior behavior and emit no CheckAlerts resource.
