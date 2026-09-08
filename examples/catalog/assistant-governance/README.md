# Assistant governance

This inert example shows a platform-owned Assistant governance request with one rule profile
selection and one MCP server allow-list. The rule text is not supplied by the request: the
Composition input resolves the `inert-guidance` profile. A platform overlay should replace that
profile with reviewed content before using the API.

Terms acceptance is a per-stack safety gate. The composition renders the `TermsAcceptance` child
first and waits for its observed `status.atProvider.accepted: true` before admitting rules or MCP
servers. Setting `spec.termsAcceptance.accepted: false` withdraws acceptance and removes the other
Assistant children on the next reconciliation, even if the old accepted state is still observed.

MCP tool approvals default to `always_ask` for named tools without an explicit policy. Keep
`auto_approve` limited to reviewed, non-destructive tools. Header values are write-only Secret data;
the example intentionally does not include a Secret reference or credential.

The catalog is not watched by Argo CD. Replace the placeholder API group, stack reference, MCP URL,
and platform rule profile in a private overlay before applying anything.
