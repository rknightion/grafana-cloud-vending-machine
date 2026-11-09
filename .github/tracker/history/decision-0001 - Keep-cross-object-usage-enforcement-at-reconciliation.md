---
id: decision-0001
title: Keep cross-object usage enforcement at reconciliation
date: '2026-09-09 09:39'
status: accepted
---
## Context

GrafanaK6Project and GrafanaSyntheticMonitoring need the referenced stack's observed usage to select and enforce platform limits. Their ValidatingAdmissionPolicy bindings already use the one available paramRef for the matching Composition. The admission mechanism cannot also read the referenced stack object. Source evidence is the single Composition paramRef in platform/apis/k6-v1beta1.yaml and platform/apis/synthetic-monitoring-v1beta1.yaml, with the observed-stack checks in platform/function/k6.go and platform/function/syntheticmonitoring.go.

## Decision

The repository owner decided on 2026-09-09 that admission enforces the request's declared values against the platform-owned Composition profile, while reconciliation binds those declarations to the referenced stack's observed usage and current cap observations. This is the shipped enforcement boundary. Existing limits do not change.

## Consequences

Admission gives an immediate refusal for declared values outside the Composition profile. A declaration that conflicts with the referenced stack, or cap resources that are absent, stale, or do not match desired state, is refused or withheld during reconciliation. Public documentation must name both timings and must not describe the managed API budget as a tenant-wide service quota.

## Alternatives

A deployed validating webhook was rejected because it adds a running TLS service, RBAC, certificate rotation, and a second copy of function policy that can drift. A generated per-stack policy lifecycle was rejected because it adds policy generation, update, cleanup, and ownership state to the composition function. Crossplane already enforces the observed-stack boundary during reconciliation, so neither added lifecycle is justified.
