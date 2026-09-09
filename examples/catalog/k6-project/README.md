# Bounded k6 project

This inert example creates one k6 project for an already-vended Grafana Cloud
stack. The platform installs k6 through the referenced stack's observed
service-account token, then creates the project, its usage-class limit set, the
selected private-load-zone allow-list, one bounded smoke load test, and one
seven-run daily schedule.

## Files

- `k6-project.yaml` requests the project, a subset of approved private load zones, one load test, and one schedule.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Prerequisites

- The referenced stack is Ready and its rotating service-account token has been observed.
- The Composition input contains a reviewed k6 limit profile for the stack's usage class.
- The requested load-zone identifiers appear in that profile. The provider can allow-list private zones, but cannot provision them.
- The declared `usage` matches the referenced stack's observed usage. It is required when load tests or schedules are requested.

## Values to replace

- Replace `platform.example.org` with your API group, the stack reference, and the Grafana user identity.
- Replace each example load-zone identifier only with one allowed by the platform profile for the referenced stack's observed usage.
- Replace the example HTTPS URL, test name, and schedule only with workload-owner-approved values that stay within the platform profile.

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

Load tests are capped by the platform profile's regular VUs, browser VUs, and
duration. The request supplies a structured HTTPS GET workload; the function
generates the bounded k6 script and sets its VUs, duration, and load-zone
distribution from the admitted fields.
Schedules wait for the provider-assigned LoadTest ID and are delete-managed
with their load tests. Removing a dynamic workload removes its LoadTest and
Schedule from desired state, and both carry `Delete` management while the
project itself retains. The function test proves this declarative deletion
boundary; it does not claim a live remote deletion was observed.

`allowedLoadZones` must be explicit: an empty array means no private load zones. Omission is rejected rather than clearing a prior set or leaving the policy unmanaged.

Admission checks the declared VUs, duration and zones against the selected
Composition profile. Admission cannot read the referenced stack. Reconciliation
therefore checks the declared usage against that stack's observed usage and
withholds dynamic children until the observed `ProjectLimits` and allowed-zone
resources match the desired profile at their current generation. The generated
script is inert in this repository: live test execution and remote schedule
deletion are not exercised by the API-server harness.
