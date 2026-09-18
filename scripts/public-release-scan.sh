#!/usr/bin/env bash
set -euo pipefail

if ! command -v rg >/dev/null 2>&1; then
  echo "public-release scan: ripgrep (rg) is required; without it the working-tree" >&2
  echo "half of every check silently no-ops and the scan reports a false pass." >&2
  exit 1
fi

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

failed=0
history_revisions=()

while IFS= read -r revision; do
  history_revisions+=("$revision")
done < <(git rev-list --all)

scan_fixed() {
  local label=$1
  local pattern=$2
  if rg --hidden --glob '!.git/**' --glob '!scripts/public-release-scan.sh' -n -i -F -- "$pattern" .; then
    echo "public-release scan: found $label in the working tree" >&2
    failed=1
  fi
  if (( ${#history_revisions[@]} > 0 )) &&
    git grep -I -n -i -F -- "$pattern" "${history_revisions[@]}" -- . \
      ':(exclude)scripts/public-release-scan.sh'; then
    echo "public-release scan: found $label in reachable Git history" >&2
    failed=1
  fi
}

scan_fixed_case_sensitive() {
  local label=$1
  local pattern=$2
  if rg --hidden --glob '!.git/**' --glob '!scripts/public-release-scan.sh' -n -F -- "$pattern" .; then
    echo "public-release scan: found $label in the working tree" >&2
    failed=1
  fi
  if (( ${#history_revisions[@]} > 0 )) &&
    git grep -I -n -F -- "$pattern" "${history_revisions[@]}" -- . \
      ':(exclude)scripts/public-release-scan.sh'; then
    echo "public-release scan: found $label in reachable Git history" >&2
    failed=1
  fi
}

scan_regex() {
  local label=$1
  local pattern=$2
  if rg --hidden --glob '!.git/**' --glob '!LICENSE' --glob '!scripts/public-release-scan.sh' \
    -n -i -- "$pattern" .; then
    echo "public-release scan: found $label in the working tree" >&2
    failed=1
  fi
  if (( ${#history_revisions[@]} > 0 )) &&
    git grep -I -n -i -E -- "$pattern" "${history_revisions[@]}" -- . \
      ':(exclude)LICENSE' ':(exclude)scripts/public-release-scan.sh'; then
    echo "public-release scan: found $label in reachable Git history" >&2
    failed=1
  fi
}

# Runs a search whose no-match status is 1 and whose error statuses are >1, so a
# broken search can never be mistaken for a clean result. Prints matches on stdout.
run_search() {
  local status=0
  "$@" || status=$?
  if (( status > 1 )); then
    echo "public-release scan: search command failed with status $status: $*" >&2
    exit 2
  fi
}

# Like scan_fixed, but tolerates occurrences that are legitimately public. Each
# hit has its allowed substrings removed before it is re-tested, so a record only
# passes when nothing forbidden survives — a line carrying both an allowed
# reference and real environment identity still fails, and an allowed substring in
# the file path does not exonerate the line's content.
sift_hits() {
  local pattern=$1
  local allowed=$2
  run_search perl -ne '
    BEGIN { $allowed = shift @ARGV; $forbidden = shift @ARGV }
    $stripped = $_;
    $stripped =~ s/$allowed//gi;
    print if $stripped =~ /\Q$forbidden\E/i;
  ' "$allowed" "$pattern"
}

# Like sift_hits, but the forbidden expression is an ERE rather than a fixed
# string. This is used only by the working-tree identifier-class control below.
sift_regex_hits() {
  local pattern=$1
  local allowed=$2
  run_search perl -ne '
    BEGIN { $allowed = shift @ARGV; $forbidden = shift @ARGV }
    $stripped = $_;
    $stripped =~ s/$allowed//gi if length $allowed;
    print if $stripped =~ /$forbidden/i;
  ' "$allowed" "$pattern"
}

# Search a working-tree root only. The existing scan helpers deliberately pair
# their working-tree search with a history search, because a finding in history
# cannot be repaired without rewriting published commits. These identifier
# classes are already present in reachable history, so pairing them would make
# this new control permanently red. Only the scan source itself is excluded;
# tests, fixtures, backlog and evidence remain in scope.
scan_working_tree_only() {
  local label=$1
  local pattern=$2
  local allowed=$3
  local scan_root=$4
  local quiet=${5:-0}
  local case_sensitive=${6:-0}
  local status
  local -a rg_args=(rg --hidden --glob '!.git/**' \
    --glob '!scripts/public-release-scan.sh' -n)
  local hits

  if (( case_sensitive == 0 )); then
    rg_args+=(-i)
  fi

  if hits=$(
    cd "$scan_root"
    if [[ -n $allowed ]]; then
      run_search "${rg_args[@]}" -- "$pattern" . |
        sift_regex_hits "$pattern" "$allowed"
    else
      run_search "${rg_args[@]}" -- "$pattern" .
    fi
  ); then
    status=0
  else
    status=$?
  fi
  if (( status > 1 )); then
    echo "public-release scan: search failed while checking $label" >&2
    return "$status"
  fi
  if [[ -n $hits ]]; then
    if (( quiet == 0 )); then
      printf '%s\n' "$hits"
    fi
    echo "public-release scan: found $label in the working tree" >&2
    return 1
  fi
}

# The fixture is deliberately created outside the repository. Keeping it out
# of the tree means the normal scan cannot pass merely because its own negative
# example is excluded, and it keeps the refused shapes out of committed files.
# Each case must fail with status 1; run_search turns a broken search (>1) into
# status 2, so a missing or malfunctioning search cannot satisfy this control.
run_refused_identifier_negative_control() (
  local fixture_dir
  local allowed_dir
  local output
  local status
  local numeric_label
  local numeric_case
  local numeric_index=0
  local synthetic_number=1
  local -a numeric_labels=(
    orgId orgID org_id org-id organizationId org
    stackId stackID stack_id stack-id 'stack id' stack
    accountId accountID account_id account-id 'account id' account
    serviceAccountId serviceAccountID service_account_id service-account-id
    'service account id' service-account service_account serviceAccount
  )

  fixture_dir=$(mktemp -d)
  allowed_dir=$(mktemp -d)
  trap 'rm -rf -- "$fixture_dir" "$allowed_dir"' EXIT
  case "$fixture_dir" in
    "$repo_root"|"$repo_root"/*)
      echo "public-release scan: negative-control fixture resolved inside the repository" >&2
      exit 2
      ;;
  esac
  case "$allowed_dir" in
    "$repo_root"|"$repo_root"/*)
      echo "public-release scan: permitted-boundary fixture resolved inside the repository" >&2
      exit 2
      ;;
  esac

  mkdir "$fixture_dir/numeric"
  for numeric_label in "${numeric_labels[@]}"; do
    numeric_case="$fixture_dir/numeric/case-$numeric_index"
    mkdir "$numeric_case"
    printf '%s: %s\n' "$numeric_label" "$synthetic_number" \
      >"$numeric_case/negative-control.md"
    numeric_index=$((numeric_index + 1))
    synthetic_number=$((synthetic_number + 1))
  done

  printf '%s\n' \
    "$(printf '%s/%s-%s\n' "$org_identifier" private repository)" \
    "$(printf '%s-%s\n' sample awsinfra)" \
    "$(printf '%s %s workshop estate\n' Example Analytics)" \
    >"$fixture_dir/negative-control.md"

  printf '%s\n' \
    'stack slug: sample-stack' \
    'region: prod-us-central-0' \
    'https://sample.grafana.net' \
    >"$allowed_dir/permitted-boundary.md"
  printf 'release: %s\n' "$synthetic_number" >>"$allowed_dir/permitted-boundary.md"

  if output=$(scan_working_tree_only \
    "refused numeric organization, stack or account identifier (negative control)" \
    "$numeric_identifier_pattern" "" "$fixture_dir/numeric/case-0" 1 2>&1); then
    echo "public-release scan: numeric identifier negative control unexpectedly passed" >&2
    exit 2
  else
    status=$?
  fi
  (( status == 1 )) || {
    echo "public-release scan: numeric identifier negative control returned status $status" >&2
    exit 2
  }
  printf '%s\n' "$output"

  for numeric_case in "$fixture_dir"/numeric/case-*; do
    if [[ $numeric_case == "$fixture_dir/numeric/case-0" ]]; then
      continue
    fi
    if scan_working_tree_only \
      "refused numeric organization, stack or account identifier (negative control)" \
      "$numeric_identifier_pattern" "" "$numeric_case" 1 >/dev/null 2>&1; then
      echo "public-release scan: numeric identifier form negative control unexpectedly passed" >&2
      exit 2
    else
      status=$?
    fi
    (( status == 1 )) || {
      echo "public-release scan: numeric identifier form negative control returned status $status" >&2
      exit 2
    }
  done

  if output=$(scan_working_tree_only \
    "refused non-allowlisted private repository name (negative control)" \
    "$private_repository_pattern" "$allowed_source_repositories" "$fixture_dir" 1 2>&1); then
    echo "public-release scan: private repository negative control unexpectedly passed" >&2
    exit 2
  else
    status=$?
  fi
  (( status == 1 )) || {
    echo "public-release scan: private repository negative control returned status $status" >&2
    exit 2
  }
  printf '%s\n' "$output"

  if output=$(scan_working_tree_only \
    "refused internal project or estate name (negative control)" \
    "$internal_project_pattern" "" "$fixture_dir" 1 1 2>&1); then
    echo "public-release scan: project or estate negative control unexpectedly passed" >&2
    exit 2
  else
    status=$?
  fi
  (( status == 1 )) || {
    echo "public-release scan: project or estate negative control returned status $status" >&2
    exit 2
  }
  printf '%s\n' "$output"

  if scan_working_tree_only \
    "permitted stack slug, region or grafana.net hostname" \
    "$numeric_identifier_pattern" "" "$allowed_dir" &&
    scan_working_tree_only \
      "permitted stack slug, region or grafana.net hostname" \
      "$private_repository_pattern" "$allowed_source_repositories" "$allowed_dir" &&
    scan_working_tree_only \
      "permitted stack slug, region or grafana.net hostname" \
      "$internal_project_pattern" "" "$allowed_dir" 0 1; then
    echo "public-release scan: permitted slug, region and grafana.net boundary controls passed."
  else
    echo "public-release scan: permitted-boundary control unexpectedly failed" >&2
    exit 2
  fi

  echo "public-release scan: refused-identifier negative controls passed."
)

scan_fixed_allowing() {
  local label=$1
  local pattern=$2
  local allowed=$3
  local hits

  hits=$(run_search rg --hidden --glob '!.git/**' \
    --glob '!scripts/public-release-scan.sh' -n -i -F -- "$pattern" . |
    sift_hits "$pattern" "$allowed")
  if [[ -n $hits ]]; then
    printf '%s\n' "$hits"
    echo "public-release scan: found $label in the working tree" >&2
    failed=1
  fi

  if (( ${#history_revisions[@]} > 0 )); then
    hits=$(run_search git grep -I -n -i -F -- "$pattern" "${history_revisions[@]}" -- . \
      ':(exclude)scripts/public-release-scan.sh' |
      sift_hits "$pattern" "$allowed")
    if [[ -n $hits ]]; then
      printf '%s\n' "$hits"
      echo "public-release scan: found $label in reachable Git history" >&2
      failed=1
    fi
  fi
}

# Source-environment identity comes from an ERE passed IN BY ENVIRONMENT, never
# from this file.
#
# This used to be a list of split string literals. Splitting stops the control
# matching itself, which is why it looked reasonable, but it does NOT hide the
# terms: it defeats grep, not a reader, and this repository is public. The
# literals sat here readable from 2026-08-04 until they were rewritten out.
#
# Accepted, in order of precedence, matching grafana-cloud-org-insights'
# bin/check-customer-identifiers so one export serves both repositories:
#   --patterns-file <path>
#   GCVM_IDENTIFIER_PATTERN                 this repository's override
#   CUSTOMER_IDENTIFIER_PATTERN             the shared set; what CI provides
#   GCINSIGHT_CUSTOMER_IDENTIFIER_PATTERN   the existing local export
#
# Absent all four this exits 2 rather than passing, because a scan that silently
# skips its identity half is worse than no scan. Extend the pattern set when a
# new engagement starts: a missing identifier means the gate quietly passes.
identity_pattern=""
patterns_file=""
allow_missing="${GCVM_SCAN_ALLOW_MISSING_PATTERNS:-0}"

while (($# > 0)); do
  case $1 in
    --patterns-file)
      patterns_file=${2:?--patterns-file needs a path}
      shift 2
      ;;
    --allow-missing-patterns)
      allow_missing=1
      shift
      ;;
    *)
      echo "public-release scan: unknown argument $1" >&2
      exit 2
      ;;
  esac
done

if [[ -n $patterns_file ]]; then
  if [[ ! -r $patterns_file ]]; then
    echo "public-release scan: --patterns-file $patterns_file is not readable" >&2
    exit 2
  fi
  identity_pattern=$(tr -d '\n' <"$patterns_file")
else
  identity_pattern="${GCVM_IDENTIFIER_PATTERN:-${CUSTOMER_IDENTIFIER_PATTERN:-${GCINSIGHT_CUSTOMER_IDENTIFIER_PATTERN:-}}}"
fi

if [[ -z $identity_pattern ]]; then
  if [[ $allow_missing == 1 ]]; then
    echo "public-release scan: WARNING - no identity pattern set, so the identity half" >&2
    echo "of this scan did not run. Only the credential and structural checks did." >&2
    echo "Acceptable only where secrets are genuinely unavailable, such as a pull" >&2
    echo "request from a fork. Never a basis for publishing anything." >&2
  else
    echo "public-release scan: no identity pattern available." >&2
    echo >&2
    echo "Set one of GCVM_IDENTIFIER_PATTERN, CUSTOMER_IDENTIFIER_PATTERN or" >&2
    echo "GCINSIGHT_CUSTOMER_IDENTIFIER_PATTERN, or pass --patterns-file <path>." >&2
    echo >&2
    echo "Exiting 2 rather than passing: a scan that skips its identity half while" >&2
    echo "reporting success is worse than no scan at all." >&2
    exit 2
  fi
else
  scan_regex "source-environment identity" "$identity_pattern"
fi

# The organisation and account names stay hardcoded deliberately. Neither is in
# the customer pattern set, and a bare org or account name discloses far less
# than a customer does. What they still catch is an environment-shaped reference
# creeping into a generic product.
#
# The allowed forms are ENUMERATED, never a bare `<org>/`. That distinction is
# the whole control: a wildcard would pass any repository under the org and the
# rule would stop meaning anything.
#
# NOTE, corrected 2026-09-11: an earlier version of this comment said each
# allowed entry is "already public". That is false - all four of the named
# repositories are private. The real bar is that Rob has judged the NAME safe to
# disclose, which is a weaker and more honest claim. Do not extend the list on
# the assumption that a repository being public is the test; ask.
#
# Longest forms first -- the alternation is tried left to right and each hit has
# its allowed substrings removed before being re-tested, so a shorter prefix
# matching first would leave the remainder behind and fail the line.
#
# `<org>/portina-iac` added 2026-09-11, and NOT because the name was judged
# publishable on its own. GCV-0068's description named it, which reached three
# commits, and this scan reads history - so the only alternatives were a
# 166-commit rewrite of a public repository or a permanently red gate. The
# rewrite was declined as disproportionate: the reference is an internal
# repository name, not customer identity, and nothing in the customer pattern
# set appears anywhere in this repository's history.
#
# The description itself was reworded to drop the qualifier, so the phrase does
# not recur. This entry exists only to cover the history that cannot be changed.
org_identifier="m7kni"
allowed_source_repositories="$org_identifier/$org_identifier-net-site|$org_identifier\\.io|$org_identifier-net-site|$org_identifier/agent-docs|$org_identifier/ci-tools|$org_identifier/renovate-config|$org_identifier/portina-iac|$org_identifier self-hosted|rknightion/$org_identifier"
scan_fixed_allowing "source API/domain identifier" "$org_identifier" \
  "$allowed_source_repositories"
scan_fixed "source account identifier" "robknight"

# This is deliberately a separate working-tree-only control. The patterns use
# shapes rather than the three historical values: organization numbers, stack
# or account numbers in an identity-labelled form, repository paths in the
# private organisation after the enumerated public references are removed, and
# proper-name phrases attached to an internal project or estate. Stack slugs,
# regions and grafana.net hostnames do not match these expressions.
# A suffix-bearing label may use camel, underscore, hyphen or spaces. A bare
# label requires punctuation, which keeps ordinary display names out while
# still catching serialized labels such as `stack: <number>`.
numeric_identifier_suffix='[[:space:]_-]?(id|identifier|number|uid)'
numeric_identifier_label='(org(anization)?|stack|account|service([_-]|[[:space:]])?account)'
numeric_identifier_pattern="(^|[^[:alnum:]_])((org(anization)?${numeric_identifier_suffix}|stack${numeric_identifier_suffix}|account${numeric_identifier_suffix}|service([_-]|[[:space:]])?account${numeric_identifier_suffix})[[:space:]]*[:#=]?[[:space:]]*[0-9]+|${numeric_identifier_label}[[:space:]]*[:#=][[:space:]]*[0-9]+)([^0-9]|$)"
private_repository_pattern="(^|[^[:alnum:]_])${org_identifier}/[[:alnum:]_.-]+([^[:alnum:]_.-]|$)|(^|[^[:alnum:]_])[a-z][a-z0-9-]*-awsinfra([^[:alnum:]_-]|$)"
internal_project_pattern='(^|[^[:alnum:]_])[[:upper:]][[:alnum:]_.-]*([[:space:]]+[[:upper:]][[:alnum:]_.-]*){1,3}[[:space:]]+((workshop[[:space:]]+)?(project|estate))([^[:alnum:]_]|$)'

if ! scan_working_tree_only \
  "refused numeric organization, stack or account identifier" \
  "$numeric_identifier_pattern" "" "$repo_root"; then
  failed=1
fi
if ! scan_working_tree_only \
  "refused non-allowlisted private repository name" \
  "$private_repository_pattern" "$allowed_source_repositories" "$repo_root"; then
  failed=1
fi
if ! scan_working_tree_only \
  "refused internal project or estate name" \
  "$internal_project_pattern" "" "$repo_root" 0 1; then
  failed=1
fi

# Architecture and proof-of-concept vocabulary from the originating engagement.
# Not customer names and not in the pattern set, so hardcoded. Splitting them
# was never needed to stop the control matching itself either - every scan
# function already excludes this file by path.
scan_fixed "source proof-of-concept identifier" "crossplaneavm"
scan_fixed "source architecture acronym" "avm"

# Matched as prefix PLUS a token body, not as the bare prefix. A real token is
# the prefix followed by a long base62 body, so this still catches every one of
# them, while prose that merely names the prefix - which an authentication bug
# report does by necessity - is not a credential and must not red-light the
# gate permanently. The scan reads reachable history, so a false positive that
# reaches a commit cannot be undone by a later one.
scan_regex "Grafana Cloud token" "gl""c_[A-Za-z0-9_-]{16,}"
scan_regex "Grafana service-account token" "gl""sa_[A-Za-z0-9_-]{16,}"
scan_fixed "private Tailscale hostname" ".ts"".net"
scan_fixed_case_sensitive "local macOS path" "/Users/"
scan_fixed_case_sensitive "private key material" "-----BEGIN ""PRIVATE KEY-----"

scan_regex "an AWS account ID, JWT-like value, or Kubernetes Secret payload" \
  'arn:aws:[a-z0-9-]+:[a-z0-9-]*:[0-9]{12}:|eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}|^[[:space:]]*kind:[[:space:]]*Secret[[:space:]]*$'
scan_regex "a private IPv4 HTTP endpoint" \
  'https?://(10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.)'

for forbidden_file in terraform.tfvars .env .envrc config.json; do
  if find . -path './.git' -prune -o -type f -name "$forbidden_file" -print | grep -q .; then
    echo "public-release scan: found forbidden file name $forbidden_file" >&2
    failed=1
  fi
done

archive_pattern='\.(7z|bin|db|dmg|exe|gz|key|kubeconfig|p12|pem|pfx|pkg|sqlite|tar|tgz|zip)$'

if git ls-files --cached --others --exclude-standard | rg -i "$archive_pattern"; then
  echo "public-release scan: found an archive, binary, key container, or local database in the working tree" >&2
  failed=1
fi

if git log --all --format= --name-only | \
  rg -i "$archive_pattern"; then
  echo "public-release scan: found a tracked archive, binary, key container, or local database in reachable Git history" >&2
  failed=1
fi

# Keep this live-control assertion in the gate. It is intentionally
# unconditional, including when --allow-missing-patterns skips the external
# identity pattern above.
run_refused_identifier_negative_control

if (( failed != 0 )); then
  exit 1
fi

echo "Public-release scan passed."
