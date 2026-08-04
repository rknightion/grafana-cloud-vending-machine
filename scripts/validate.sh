#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

./scripts/public-release-scan.sh

if [[ -n $(gofmt -l platform/function/*.go) ]]; then
  echo "Go source is not formatted:" >&2
  gofmt -l platform/function/*.go >&2
  exit 1
fi

(
  cd platform/function
  go mod tidy
  git diff --exit-code -- go.mod go.sum
  go test -race -cover ./...
  go vet ./...
)

find . -type f \( -name '*.yaml' -o -name '*.yml' \) -not -path './.git/*' -print0 |
  xargs -0 ruby -ryaml -e 'ARGV.each { |path| YAML.parse_stream(File.read(path)) }'

kubectl kustomize platform >/dev/null
kubectl kustomize deploy/aws >/dev/null
kubectl kustomize examples/catalog/comprehensive >/dev/null
kubectl kustomize examples/catalog/minimal >/dev/null

echo "Validation passed."
