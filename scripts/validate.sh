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

test -f examples/README.md
for example_dir in examples/catalog/*; do
  if [[ -d "$example_dir" && ! -f "$example_dir/README.md" ]]; then
    echo "Missing example README: $example_dir/README.md" >&2
    exit 1
  fi
done

ruby -ryaml -e '
  # The signature-verification jobs carry their digest in argv, and the package
  # resources carry it in spec.package. Nothing links the two, so a package bump
  # that forgets the job leaves a green verification of the previous digest.
  def digests(path)
    documents = YAML.load_stream(File.read(path)).compact
    job = documents.find { |d| d["kind"] == "Job" }
    package = documents.find { |d| %w[Function Provider].include?(d["kind"]) }
    abort "#{path}: expected one verification Job and one package resource" if job.nil? || package.nil?
    args = job.dig("spec", "template", "spec", "containers", 0, "args") || []
    [args.grep(/@sha256:/).first, package["spec"]["package"]]
  end

  ["platform/function/install.yaml", "platform/provider/provider-grafana.yaml"].each do |path|
    verified, installed = digests(path)
    abort "#{path}: verification job has no digest argument" if verified.nil?
    next if verified == installed
    abort "#{path}: verifies #{verified} but installs #{installed}"
  end
'

ruby -ryaml -e '
  document = YAML.safe_load(File.read("deploy/argocd/requests-applicationset.yaml"))
  directories = document.dig("spec", "generators", 0, "git", "directories")
  abort "ApplicationSet must watch only top-level enabled/*" unless directories == [{"path" => "enabled/*"}]
'

echo "Validation passed."
