# Agent Observability module

This inert example opts into the `GrafanaAgentObservability` module for one
referenced stack. It keeps platform-owned request guards under `spec.guards`
and workload-owned collections, evaluators, and evaluation rules under
`spec.workload`.

The guard and evaluator bodies are placeholders only. Replace every example
name, stack reference, matcher, evaluator definition, and collection policy
with reviewed workload or platform values before use. Collection IDs are
assigned by Grafana; the composition waits for the observed Collection ID
before creating a RuleAction.

This module does not infer workload resources from a stack request, install
the Agent Observability plugin, or provide credentials. The referenced stack
must already expose the per-stack ProviderConfig, and the plugin and required
permissions are environment-owned prerequisites.
