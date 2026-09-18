# Lane E documentation packet

Target: `docs/diagrams/decommission-lifecycle.html`
Anchor: the visible readiness note beginning `Readiness: observed deleteProtection=false`.

Replace that note with this finished prose:

> Readiness: observed deleteProtection=false; ESO finalizes and syncs current-generation PushSecrets. Git Sync retains its external Connection and Securevalue by default. An authorized Delete records observed controller preparation and a parent-status witness before removing Connection, then repeats the evidence sequence for Securevalue; `status.decommission.phase` records the result through Complete.

Target: `docs/diagrams/decommission-lifecycle.html`
Anchor: `decommission-lifecycle-desc`.

Replace that description with this finished prose:

> State diagram showing the retain-default exit and the separately reviewed deletion stages. Git Sync credential decommission is platform-authorized, records an observed managed-resource preparation and parent-status witness before each removal, removes the Connection before the Securevalue, and records completion through claim status instead of inferred child disappearance.
