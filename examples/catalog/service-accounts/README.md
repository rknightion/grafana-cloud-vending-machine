# Bounded in-stack service accounts

This inert example requests one named in-stack Grafana service account for an
already-vended stack. The request can select only a profile and account names.
The enforced Composition owns the basic role, the complete permission set, the
rotating-token lifetime, and the token rotation window.

## Files

- `service-accounts.yaml` selects the `ci` profile and requests one account.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Prerequisites

- The referenced stack is Ready in the same namespace.
- The enforced Composition contains the selected service-account profile.
- The selected profile's token lifetime does not exceed the same
  `maximumTokenLifetime` used by the stack Composition.

## Values to replace

- Replace the API group and stack reference with values from your platform.
- Replace the profile only with one configured by the enforced Composition.
- Replace account names with DNS-compatible names for the consuming workload.

## Credentials and permissions

The renderer waits for Grafana to observe the service-account ID before it
creates the rotating token and the one `ServiceAccountPermission` whole-set
owner. It never emits `ServiceAccountPermissionItem`, because an item writer
would compete with that owner and leave unmanaged grants in place. The provider
writes the generated token to its connection Secret; a `PushSecret` publishes
only the derived `service_account_token` document to the configured secret
store. No token value is present in this request, its status, or this example.

The request name must equal `stackRef.name`, and the stack reference is immutable. This establishes one declarative owner for this surface per stack.
