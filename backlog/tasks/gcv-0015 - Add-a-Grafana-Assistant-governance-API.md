---
id: GCV-0015
title: Add a Grafana Assistant governance API
status: To Do
assignee: []
created_date: '2026-08-21 12:14'
labels: []
dependencies:
  - GCV-0010
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The Assistant resources are a policy surface a platform team can own declaratively, verified against the provider schemas:

- assistant Rule carries ruleContent injected into the Assistant system prompt, a scope of tenant or user, a priority where lower applies first, and an applications list including assistant, loop and infrastructure_memory.
- assistant MCPServer controls which external tool servers the Assistant may call, with toolApprovalPolicies keyed by tool name taking auto_approve or always_ask, plus customHeaders and an enabled flag.
- assistant TermsAcceptance is a per-stack singleton whose accepted boolean gates Assistant usage; setting it false withdraws acceptance.

Together these let the platform ship standing instructions, an allow-list of reachable tool servers, mandatory human approval on destructive tools, and an org-wide off switch.

Ordering trap: terms acceptance gates the rest and nothing enforces that dependency automatically, so the function must gate on it the way access claims already gate on stack readiness.

TermsAcceptance is one of the ten kinds absent from the current pin, hence the dependency.

Do not put rule text in this repository beyond inert examples; real standing instructions are environment-owned.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Terms acceptance is rendered and gated before any other Assistant child is admitted
- [ ] #2 MCP server entries carry per-tool approval policies and default to the more restrictive option where unspecified
- [ ] #3 Rule content is selected from platform-owned configuration, with only inert examples in this repository
- [ ] #4 Withdrawing acceptance is a supported, documented path rather than an accident
- [ ] #5 A catalog example renders with a rule and an MCP server allow-list
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
