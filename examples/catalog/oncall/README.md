# On-call schedules and alert delivery

This inert example creates a rotating on-call schedule, an escalation chain, and an inbound-email integration for one existing stack. Keep `metadata.name` equal to `spec.stackRef.name`, then replace that stack name, responder username, and shift start with approved values before use. The example start is deliberately historical because the catalog is inert.

The composition looks up each declared OnCall user and waits for the provider to observe its assigned identity before it creates the shift. `shiftStart` uses the pinned provider's `YYYY-MM-DDTHH:MM:SS` format without a zone suffix and is interpreted in the UTC schedule. It then waits for each provider-assigned shift, schedule, chain, and integration identity before emitting the dependent resource. The final route is an exhaustive catch-all for alerts received by the integration.

The released `oncall.yaml` omits `integrationType` and Slack selection: its inbound-email identities and address-based contact-point behavior are unchanged.

## Grafana Alerting to IRM and Slack

`irm.yaml` is a second, independent stack example with a linked claim set: a `grafana_alerting` Integration, the schedule-backed EscalationChain, an IRM ContactPoint selected through `onCallRef`, and a regex notification-policy route. Replace the stack name in both requests, the responder, and the Slack display name with approved values before use. Do not apply both OnCall examples to the same stack: one request owns its catch-all route, and one routing request owns its whole notification-policy tree.

Select at most one of `route.channelRef.name` (observe-only display-name lookup) or `route.channelId` (opaque Slack ID). Direct IDs must match the pinned official Slack patterns; display names, whitespace, and malformed IDs are refused. Syntax alone does not prove a channel exists. The function waits for current lookup and Route observations to confirm the exact enabled destination, and reports a fixed warning while destination readback is missing or mismatched. Removing a selection clears the desired Slack list and requires no enabled destination in readback before readiness.

For Grafana Alerting, the control plane validates the current Integration and its HTTPS link. The ContactPoint carries only a Required/Always Integration reference; the pinned provider extracts the link from that Integration. Never add a webhook URL to a consumer manifest. Controller-managed Integration status and provider-managed transport remain sensitive control-plane surfaces; this is not a provider-wide redaction guarantee.

Type changes use distinct Integration, Route, and Slack-lookup identities with a controller-owned non-secret `status.onCallIdentity` journal. Both replacement children must be observed current before the predecessor Route, then Integration, relinquish Kubernetes ownership. External resources are retained by the existing management policies. Reverse changes adopt inventoried identities. Request edits during an in-flight transaction are deferred; a committed transaction finishes its frozen configuration before applying them. Closed stack admission freezes advancement and preserves trusted applied children. Do not edit or remove the journal to force progress.

All examples are inert, are not watched by Argo CD, and make no live request on their own. Offline renderer, schema, and ephemeral API-server checks do not prove provider reconciliation, adoption, notification delivery, or migration: those are not exercised, and not exercisable here.
