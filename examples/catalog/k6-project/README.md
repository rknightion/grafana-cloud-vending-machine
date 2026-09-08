# Bounded k6 project

This inert example creates one k6 project for an already-vended Grafana Cloud
stack. The platform installs k6 through the referenced stack's observed
service-account token, then creates the project, its usage-class limit set, and
the selected private-load-zone allow-list.

## Files

- `k6-project.yaml` requests the project and a subset of approved private load zones.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Prerequisites

- The referenced stack is Ready and its rotating service-account token has been observed.
- The Composition input contains a reviewed k6 limit profile for the stack's usage class.
- The requested load-zone identifiers appear in that profile. The provider can allow-list private zones, but cannot provision them.

## Values to replace

- Replace `platform.example.org` with your API group, the stack reference, and the Grafana user identity.
- Replace each example load-zone identifier only with one allowed by the platform profile for the referenced stack's observed usage.

## Limits and credentials

Requests cannot set or raise `vuhMaxPerMonth`, `vuMaxPerTest`,
`vuBrowserMaxPerTest`, or `durationMaxPerTest`. The renderer selects all four
limits from the platform-owned profile for the referenced stack's usage. A
request can only select a subset of that profile's private zones.

The bootstrap input is never persisted in this request or its published
credential. The provider installation uses the referenced stack's canonical
token Secret and writes a derived k6 token. Only that derived token is mirrored
to the configured secret store. An ExternalSecret materializes a JSON credential
containing only `k6_access_token`, and a dedicated namespaced ProviderConfig uses
it for the Project, ProjectLimits, and ProjectAllowedLoadZones resources. The
organization ProviderConfig is used only for the installation exchange.

This API deliberately vends no k6 load tests or schedules. Test scripts,
targets, execution cadence, and load shape require workload-owner decisions and
are outside a stack request.

`allowedLoadZones` must be explicit: an empty array means no private load zones. Omission is rejected rather than clearing a prior set or leaving the policy unmanaged.
