# Cloud integrations and bounded scrape jobs

This inert catalog base installs integrations and account configuration selected by a named platform profile, then requests a complete set of CloudWatch, resource-metadata, or HTTPS metrics-endpoint scrape jobs for one ready stack.

## Values to replace

- Replace `platform.example.org` and every `replacewithunique09` occurrence. `metadata.name` must equal `spec.stackRef.name` so this composite is the sole owner of the stack integration set.
- Select a profile approved for the referenced stack. Profiles hold cloud account configuration and credential references; requests do not accept cloud roles, client secrets, or credential values.
- Replace service names, metrics, endpoint URL, and Secret reference with reviewed values. The Secret reference names a pre-existing local Secret and never puts its value in this request.

## Budget and ownership

The platform profile sets the maximum job count and the minimum scrape interval. Admission policy and the renderer both reject requests above that count or below that interval. `scrapeJobs` is the whole set owned by this composite; removing an item withdraws the matching managed scrape job.

These limits apply only to resources created through this API. Namespace RBAC must deny direct provider resource creation when the platform budget is a governance boundary.

## Files

- `cloud-integrations.yaml` is an inert request with synthetic identifiers and an invalid endpoint.
- `kustomization.yaml` makes this directory directly renderable with Kustomize.
