# Wave 14 Lane C documentation packet

Target: `docs/reference/request-schema.md`, under the `GrafanaStackConsumer` request section after
the request-field table.

Finished prose:

> The shipped `robk-telemetry-writer` profile admits requests only from the
> `telemetry-consumers` namespace for the `robk` stack in `prod-gb-south-1`; the request name and
> profile must match. The profile renders a credential publication at
> `/platform/grafana-cloud/consumers/robk-telemetry` whose `telemetry.json` contains `stack_slug`,
> `stack_region`, `access_policy_name`, and the vended `access_policy_token`. It grants
> `metrics:write`, `logs:write`, and `traces:write` because the consumer publishes those three
> telemetry signals. Live secret-store read-back is not exercised, and not exercisable here; the
> consuming estate verifies its approved store after deployment.
