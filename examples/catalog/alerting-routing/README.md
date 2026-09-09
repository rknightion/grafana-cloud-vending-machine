# Alerting routing

This inert catalog example owns the complete Grafana notification policy tree for one stack and
creates the email contact points named by that tree. Replace the stack placeholder before use.

The request name must equal `stackRef.name`, so one namespace has one declarative owner for a
stack's tree. The default contact point receives unmatched alerts. Each other vended contact point
must appear in a route, which admission enforces.

The provider replaces policy nodes omitted from `routes`; content in the provider-managed policy
tree that this request does not vend is removed on reconciliation. Grafana rule groups that set a
direct contact point use a separate mechanism and are not part of this tree.

`ruleGroups` emits ordinary Grafana alert rules without `notificationSettings`, so matching alerts
traverse this request's notification policy tree. Its names are prefixed independently from the
older alerting bundle and therefore do not share remote rule-group identities.

The operations contact point selects `onCallRef.name`. Install the matching
GrafanaOnCall request from the oncall catalog in the same namespace and stack.
The function waits for its observed integration email; no address is copied by
a human. The audit contact point demonstrates a literal email destination.
A contact point must select exactly one destination form.

The end-to-end tests admit and read back the configured graph at a real API
server with labelled provider observations. They do not evaluate rules, send
SMTP or page a live responder. Existing GrafanaAlertingBundle rules retain
per-rule direct routing and are separate from this policy-routed path.
