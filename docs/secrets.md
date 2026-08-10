---
title: Secrets
description: How External Secrets Operator wires the organization credential in and rotating per-stack tokens out
---

# Secrets

No Grafana credential belongs in Git, a request object, Composition input, status, or function
log.

## Organization credential

Create a Grafana Cloud access policy token with only the organization-level capabilities needed
to manage stacks and their Cloud resources. Store it in the external secret manager as JSON:

```json
{
  "cloud_access_policy_token": "REPLACE_SECURELY"
}
```

The example expects this document at `/platform/grafana-cloud/organization/credentials`. Do not
put the real token directly in a shell command, terminal history, CI variable dump, or Kubernetes
manifest — use your secret-management workflow or a permission-restricted temporary file.

`deploy/aws/secret-store-and-credentials.yaml` wires this in three pieces:

1. A `SecretStore` (`grafana-vending-secrets`) pointing at AWS Secrets Manager.
2. An `ExternalSecret` (`grafana-cloud-org-credentials`) that reads the token from
   `/platform/grafana-cloud/organization/credentials` and templates it into the JSON shape the
   provider expects, refreshed hourly.
3. A namespaced `ProviderConfig` (`grafana-cloud-org`) that references the generated Kubernetes
   Secret and is used for all organization-level Cloud operations.

Namespace RBAC should prevent request authors from reading the generated Secret directly.

## Rotating administrator token

For every stack, the Composition creates:

1. A `StackServiceAccount` with the Admin role.
2. A `StackServiceAccountRotatingToken`, created after Grafana reports the service-account ID —
   this is why a brand-new request needs at least two reconciliation passes before its
   credentials exist.
3. A Kubernetes connection Secret written by the rotating-token resource.
4. A `PushSecret` that exports a structured administrator document.
5. An `ExternalSecret` that reads the exported token and URL back into the stack namespace.
6. A stack-local `ProviderConfig` used for Grafana resources inside that stack.

The token lifetime is 30 days with a seven-day early rotation window. The `PushSecret` refresh
interval is one hour, so a newly rotated token is copied to the external store well inside the
overlap window.

The exported document has this shape:

```json
{
  "stack_name": "Example stack",
  "stack_slug": "example",
  "stack_url": "https://example.grafana.net",
  "stack_region": "prod-us-central-0",
  "usage": "development",
  "change_reference": "CHANGE-EXAMPLE",
  "configuration_item_reference": "CONFIG-EXAMPLE",
  "stack_service_account_token": "GENERATED",
  "telemetry_access_policy_secret_path": "/platform/grafana-cloud/stacks/REGION/USAGE/SLUG/telemetry-publisher"
}
```

The generated token comes from the connection Secret at reconciliation time; it is never
embedded in rendered YAML.

## Rotating telemetry token

When `spec.telemetryAccess.enabled` is `true` (the default), the Composition creates a
stack-realm `AccessPolicy` with only:

- `stacks:read`
- `metrics:write`
- `logs:write`
- `traces:write`

An `AccessPolicyRotatingToken` uses the same 30-day lifetime and seven-day early rotation window.
A separate `PushSecret` publishes the token and policy metadata under the stack's
`telemetry-publisher` path. Workloads should use this token for telemetry and never receive the
administrator token.

Static `StackServiceAccountToken`, `AccessPolicyToken`, and `ServiceAccountToken` resources
remain available in the upstream provider but are deliberately not used here — their rotating
counterparts avoid creating a permanent credential lifecycle outside the control plane.

## SSO and incident profile secrets

`deploy/aws/optional-profile-secrets.yaml` provides the `ExternalSecret` shape for OAuth client
secrets and the incident relay authorization value, each read from its own remote path under
`/platform/grafana-cloud/profiles/<profile-name>`. These are intentionally excluded from
`deploy/aws/kustomization.yaml` — apply this file only after the corresponding remote secrets and
profile definitions exist, since an `ExternalSecret` referencing a missing remote value fails to
sync. See [SSO](sso.md) for how a request selects a profile.

## AWS permissions

`deploy/aws/iam-policy.json` is the minimum policy shape for the example paths:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ReadVendingMachineInputsAndOutputs",
      "Effect": "Allow",
      "Action": ["secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"],
      "Resource": ["arn:aws:secretsmanager:*:*:secret:/platform/grafana-cloud/*"]
    },
    {
      "Sid": "CreateAndRotateVendingMachineOutputs",
      "Effect": "Allow",
      "Action": ["secretsmanager:CreateSecret", "secretsmanager:PutSecretValue", "secretsmanager:TagResource"],
      "Resource": ["arn:aws:secretsmanager:*:*:secret:/platform/grafana-cloud/stacks/*"]
    }
  ]
}
```

Attach it to the `external-secrets` ServiceAccount through the workload-identity mechanism for
your cluster:

- **EKS Pod Identity** — create a Pod Identity association for namespace `external-secrets` and
  ServiceAccount `external-secrets`.
- **IRSA** — annotate that ServiceAccount with its role and use the standard EKS OIDC trust
  relationship.
- **Other Kubernetes platforms** — use the cloud identity integration recommended for that
  platform.

Do not put long-lived AWS keys in the `SecretStore`. The controller should obtain short-lived
credentials from workload identity. Restrict `CreateSecret`, `PutSecretValue`, and `TagResource`
to the output prefix; restrict read access to the input and output paths actually required.

The example uses AWS Secrets Manager, not Systems Manager Parameter Store. The composition
function itself only emits `SecretStore` references, so another ESO provider can be substituted
if it supports `ExternalSecret` and `PushSecret` with the required structured-value behaviour.

`PushSecret` uses `deletionPolicy: None`. Removing a request does not delete its external
credential documents — see [Architecture → baseline and optional resources](architecture.md) and
the decommission runbook in the project
[README](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook) for
the intentional non-destructive lifecycle this produces.

## Extra composition RBAC

`platform/rbac/composition-rbac.yaml` grants the Crossplane composition RBAC manager an
aggregated `ClusterRole` (`crossplane-compose-grafana-vending-secrets`) with full verbs on
`pushsecrets` and `externalsecrets` (and their `/status` subresources). Crossplane needs this
because the composition function emits `PushSecret` and `ExternalSecret` objects as part of a
stack's composed resources, which is outside Crossplane's default RBAC surface.

## Next steps

- [SSO](sso.md) — how a request selects a platform-owned identity profile.
- [Security](security.md) — supply-chain verification and the non-destructive lifecycle.
- [Installation](installation.md) — where the `SecretStore` and organization `ProviderConfig` are
  applied during bootstrap.
