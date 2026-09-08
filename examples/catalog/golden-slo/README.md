# Golden SLO by usage class

This inert catalog base pairs a production stack request with platform-owned `goldenSLOProfiles`
configuration. A selected profile creates one ratio SLO after the named destination datasource has
been observed. Grafana generates the recording rules and the fast-burn, slow-burn, and remaining
budget alerts from that SLO.

The profile belongs in the platform function input, not in a stack request. The public metric names
and datasource UID are placeholders; replace them in an environment overlay with an existing
Prometheus-compatible datasource and metrics that measure a real workload.

The SLO is created through `initProvider`, so the platform supplies an initial template and later
tenant edits are retained. The template deliberately supplies no alert enrichment because its
assistant-investigation surface is preview and may change incompatibly.
