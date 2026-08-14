# GitHub Issues archive

`issues-dump.json` is the complete, verbatim archive of this repository's GitHub Issues as they
stood on 2026-08-14, captured immediately before the issues themselves were deleted. It is the
record, not a convenience copy: the issues no longer exist on GitHub, so `gh issue view <N>` will
not resolve any of them.

The archive was committed and pushed **before** anything was deleted, and every number in the delete
list was asserted present here first. The Issues tab remains enabled for external contributors.

Three issues, all closed as completed, all authored by the repository owner: `#1`, `#3`, `#4`.
(`#2` was a pull request and is not part of the issue archive.) Every issue body and every comment
is present — 11 comments in total, matching the API's own per-issue comment counts exactly, which
was checked rather than assumed because the issue-list JSON paginates comments.

Work tracking moved to Backlog.md in the same change. The **Closed GitHub issues — pre-Backlog
history index** document in `backlog/docs/` is the readable index of what these issues were and
which commits closed them; this file is the full text behind it.

## Reading it

```bash
# titles, states and closing dates
jq -r '.[] | "#\(.number) [\(.state)] \(.closedAt // "open") — \(.title)"' archive/issues-dump.json

# one issue in full, body then comments
jq -r '.[] | select(.number == 4) | .body, "\n--- comments ---", (.comments[] | "\n[\(.createdAt)]\n\(.body)")' \
  archive/issues-dump.json
```

## Redaction

**Nothing was redacted, and no placeholder mapping is needed** — this archive is verbatim.

That is a finding, not an omission. This repository was written from the outset as a portable public
reference whose issues deliberately carried no source-environment identity, and `scripts/public-release-scan.sh`
enforces that on every commit. Before the archive was committed, its **decoded** fields — not its
serialized JSON — were swept against every pattern that scan rejects, and against email addresses,
account and stack identifiers, private hostnames, IP addresses and token prefixes. Nothing matched.

The decoded-versus-serialized distinction is load-bearing. In `json.dumps` output an escape such as
`\n` leaves a literal `n` immediately before the following word, which breaks a `\b` word boundary
and lets a sweep of the raw file certify as clean a document that still leaks. Any future sweep of
this file must walk the parsed structure.

The one identifier that does appear throughout is `rknightion`, the repository owner, in issue and
comment URLs. It is the public account this repository already lives under and is not redacted.
