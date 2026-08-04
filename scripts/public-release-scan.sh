#!/usr/bin/env bash
set -euo pipefail

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

# These strings are split so the safety control does not reproduce source
# environment identifiers in a repository intended for publication.
scan_fixed "source customer identifier" "ro""che"
scan_fixed "source API/domain identifier" "m7""kni"
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
