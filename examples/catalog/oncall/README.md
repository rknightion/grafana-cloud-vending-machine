# On-call schedules and alert delivery

This inert example creates a rotating on-call schedule, an escalation chain, and an inbound-email integration for one existing stack. Keep `metadata.name` equal to `spec.stackRef.name`, then replace that stack name, responder username, and shift start with approved values before use. The example start is deliberately historical because the catalog is inert.

The composition looks up each declared OnCall user and waits for the provider to observe its assigned identity before it creates the shift. `shiftStart` uses the pinned provider's `YYYY-MM-DDTHH:MM:SS` format without a zone suffix and is interpreted in the UTC schedule. It then waits for each provider-assigned shift, schedule, chain, and integration identity before emitting the dependent resource. The final route is an exhaustive catch-all for alerts received by the integration.

The alerting-side contact point is supplied by the platform adapter after the integration publishes its inbound email address. This example is not watched by Argo CD and makes no live request on its own.
