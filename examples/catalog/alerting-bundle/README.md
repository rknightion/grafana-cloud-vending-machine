# Alerting bundle

This inert catalog example defines stack-scoped Grafana alerting objects. Replace the stack and
folder placeholders before using it. The `example.invalid` recipient is deliberately non-routable.

`provenance` is required. `enforced` keeps the provisioning provenance marker, so tenant Grafana UI
edits are locked. `createOnly` sets the provider's disable-provenance flag and seeds values through
`initProvider`, so later tenant UI edits are permitted and are not reconciled back by this bundle.

The rule group uses simplified per-rule routing through `notificationSettings`. It deliberately does
not create a `NotificationPolicy` or `RoutingtreeV1Beta1`: a notification policy owns Grafana's
organisation-wide tree and deletion resets that tree to defaults. Per-rule routing keeps this bundle
inside its stack boundary without taking ownership of a whole-tree singleton.

Every object name is prefixed with the bundle's Kubernetes name. The bundle is therefore the sole
owner of its rule groups, contact points, mute timings, message templates, and inhibition rules.
The older optional stack contact-point path keeps owning only its incident relay contact points;
because this bundle prefixes its own contact-point names, the two paths do not address the same
Grafana object.
