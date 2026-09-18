# Lane D control evidence

Task: GCV-0086

The working-tree-only helper scans every repository path except its own source
file. Its deliberate fixtures are generated under temporary directories outside
the repository. Numeric labels are generated from shape names and a runtime
counter, so no refused instance is committed. The numeric control tests each
org, stack, account and service-account label separately, then the private-name
and internal-name controls run against their own generated fixture. A separate
boundary fixture contains a stack slug, a region, a `grafana.net` hostname and
an unrelated release number; all three refused-shape checks pass over it.

The following is the verbatim output from `just public-release-scan` on the
current working tree. The first three lines are the expected failures observed
inside the deliberate negative control; the control then reports its permitted
boundary checks and the enclosing scan passes.

```text
./scripts/public-release-scan.sh
public-release scan: found refused numeric organization, stack or account identifier (negative control) in the working tree
public-release scan: found refused non-allowlisted private repository name (negative control) in the working tree
public-release scan: found refused internal project or estate name (negative control) in the working tree
public-release scan: permitted slug, region and grafana.net boundary controls passed.
public-release scan: refused-identifier negative controls passed.
Public-release scan passed.
```

Additional validation:

- `bash -n scripts/public-release-scan.sh` passed.
- Running with no `rg`, including `--allow-missing-patterns`, exited one and
  printed the loud missing-tool message.
- A deliberately broken `rg` exited two through `run_search`, proving search
  errors are not treated as no-match results.
- With all identity-pattern variables unset and `--allow-missing-patterns`, the
  scan still ran the negative control and passed its working-tree checks.
- `backlog doc update doc-0002 --content ...` completed, and the CLI readback
  matched the submitted complete document body.
- `just check` was not run; it is owned by the root integration pass.
