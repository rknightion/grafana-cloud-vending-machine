# Fleet Management baseline pipelines

This inert example selects the platform-owned `standard` Fleet Management baseline for a previously vended stack. It writes Pipelines only. Collectors self-register and are deliberately observed rather than created or reconciled by the platform.

## Files

- `fleet-pipelines.yaml` selects the Fleet baseline by profile and refers to the stack request.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Prerequisites

- The referenced stack request is Ready and its per-stack provider credential publication includes the separately minted Fleet Management basic-auth value.
- Fleet Management is available for the target stack and the platform profile has been reviewed for its matcher and attribution policy.

## Values to replace

- Replace `platform.example.org` with your API group.
- Replace `replacewithunique01` with the name of the already-vended stack request.
- Replace `standard` only with an approved platform-owned Fleet pipeline profile. Do not add pipeline contents, matchers, credentials, or attribution values to this request.

## Limitations

The baseline pipeline enforces team, cost-centre, and environment labels, but it does not create usage groups. Usage groups are configured only in the Grafana UI and require the Advanced tier. The labels make later UI-managed chargeback grouping possible; they do not make that grouping declarative.
