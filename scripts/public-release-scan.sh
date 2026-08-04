#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

failed=0

scan_fixed() {
  local label=$1
  local pattern=$2
  if rg --hidden --glob '!.git/**' --glob '!scripts/public-release-scan.sh' -n -i -F -- "$pattern" .; then
    echo "public-release scan: found $label" >&2
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
scan_fixed "local macOS path" "/Users/"
scan_fixed "private key material" "-----BEGIN ""PRIVATE KEY-----"

if rg --hidden --glob '!.git/**' --glob '!LICENSE' -n \
  'arn:aws:[a-z0-9-]+:[a-z0-9-]*:[0-9]{12}:|eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}|^[[:space:]]*kind:[[:space:]]*Secret[[:space:]]*$' .; then
  echo "public-release scan: found an AWS account ID, JWT-like value, or Kubernetes Secret payload" >&2
  failed=1
fi

for forbidden_file in terraform.tfvars .env .envrc config.json; do
  if find . -path './.git' -prune -o -type f -name "$forbidden_file" -print | grep -q .; then
    echo "public-release scan: found forbidden file name $forbidden_file" >&2
    failed=1
  fi
done

if (( failed != 0 )); then
  exit 1
fi

echo "Public-release scan passed."
