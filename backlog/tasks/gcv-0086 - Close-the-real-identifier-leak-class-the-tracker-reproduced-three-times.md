---
id: GCV-0086
title: Close the real-identifier leak class the tracker reproduced three times
status: Done
assignee: []
created_date: '2026-09-18 10:55'
updated_date: '2026-09-18 15:54'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 86000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Real account identifiers reached this public repository's committed tracker markdown three times, and the third time was not caught by anything. GCV-0068's description named a private repository, which reached three commits and was resolved at scripts/public-release-scan.sh by allowlisting the literal and rewording the description. GCV-0074 then carried a numeric stack id and a numeric service-account id from 2026-09-12. GCV-0085 then carried a numeric org id, a numeric stack id, a private repository name and an internal project name from 2026-09-18. The hosted gate passed on every one of those commits.

WHY THE SCAN DID NOT CATCH THEM. Every check in public-release-scan.sh runs against the working tree AND against every reachable revision from git rev-list --all. That property is the reason a leak here is permanent, and it is also the reason the obvious fix does not work: adding these literals to the pattern set turns the gate permanently red, because the literals are already in history and this repository does not rewrite published history. The existing allowlist-plus-reword route only scales to a named repository, not to an open class of numeric identifiers.

THE FROZEN OWNER DECISION, 2026-09-18. Stack slugs, regions and grafana.net hostnames are publishable in this repository. Bare numeric org ids, stack ids and account ids are not, and neither are private repository names outside the existing enumerated allowlist or internal project and estate names. That boundary is narrower than AGENTS.md's current blanket prohibition on stack slugs, and AGENTS.md has to be corrected to match rather than left contradicting the shipped control.

THE SHAPE OF THE CONTROL. The scan needs a working-tree-only mode for this class: a pattern set that fails on the working tree and does not consult history, so a new paste fails the gate on the commit that introduces it while the existing three leaks stay green. Every current helper couples the two halves, so this is a new helper rather than a new pattern in an existing one. Follow scripts/refused-shapes.sh for the negative-control pattern: a deliberate fixture that must fail, proving the check is live rather than silently matching nothing.

The working-tree scrub of GCV-0074 and GCV-0085 was already applied by the wave 14 root before dispatch, so the check can be added against a clean tree. GCV-0085's filename still ends in a stack slug, which is permitted content under the decision above.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 public-release-scan.sh gains a working-tree-only check that rejects bare numeric org, stack and account ids, private repository names outside the enumerated allowlist, and internal project or estate names, and that check does not consult reachable history
- [x] #2 The check is proven live by a deliberate negative fixture that fails with the recorded message, in the manner of scripts/refused-shapes.sh, and the control is invoked by the gate rather than only runnable by hand
- [x] #3 The working tree passes the new check at the completing SHA, and the existing history-reading checks still pass, so the gate is green end to end
- [x] #4 AGENTS.md and the Wave operating model document are corrected to state the frozen boundary: slugs, regions and grafana.net hostnames permitted; bare numeric ids and private repository or project names refused
- [x] #5 The three prior leaks are recorded in one place with the reason history was not rewritten, so a later agent does not re-open the rewrite question without new evidence
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 14: add a working-tree-only identifier-class control with a failing negative fixture, correct the repository policy documents, and preserve the existing history scan.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 14 added the unconditional working-tree identifier control, external failing fixtures, permitted-boundary controls and loud search-failure propagation. The repository policy and Wave operating model now match the frozen boundary. The integrated local gate and exact-SHA hosted validation passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Completed at 1f2af4a0b37c244a799e1fe0cdfcea2fa5abd82e; hosted Validate public reference run 35364595851 passed at that exact SHA. Refused numeric identifiers, non-allowlisted private names and internal project or estate names now fail the gate without rewriting history.
<!-- SECTION:FINAL_SUMMARY:END -->
