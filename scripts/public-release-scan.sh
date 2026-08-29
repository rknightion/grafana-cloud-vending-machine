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

# These strings are split so the safety control does not reproduce source
# environment identifiers in a repository intended for publication.
scan_fixed "source customer identifier" "ro""che"
# The organisation name is also the public documentation hub this repository
# publishes into, so hub references pass and environment identity still fails.
#
# The allowed forms are ENUMERATED, never a bare `<org>/`. That distinction is
# the whole control: a wildcard would pass any repository under the org and the
# rule would stop meaning anything. Each entry below is a named repository or
# domain that is already public, or a phrase that names the org without naming
# an environment.
#
# Longest forms first -- the alternation is tried left to right and each hit has
# its allowed substrings removed before being re-tested, so a shorter prefix
# matching first would leave the remainder behind and fail the line.
#
# Added 2026-08-29: the backlog task in this repo (GCV-0031, one of a fleet-wide
# set) names the shared CI tooling and Renovate config repositories and the
# self-hosted runner pool. Rob confirmed those four forms are publishable.
org_identifier="m7""kni"
scan_fixed_allowing "source API/domain identifier" "$org_identifier" \
  "$org_identifier/$org_identifier-net-site|$org_identifier\\.io|$org_identifier-net-site|$org_identifier/agent-docs|$org_identifier/ci-tools|$org_identifier/renovate-config|$org_identifier self-hosted|rknightion/$org_identifier"
scan_fixed "source account identifier" "rob""knight"
scan_fixed "source proof-of-concept identifier" "crossplane""avm"
scan_fixed "source architecture acronym" "a""vm"
scan_fixed "Grafana Cloud token prefix" "gl""c_"
scan_fixed "Grafana service-account token prefix" "gl""sa_"
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

if (( failed != 0 )); then
  exit 1
fi

echo "Public-release scan passed."
