#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

./scripts/public-release-scan.sh

ruby -ryaml -e '
  # platform/kustomization.yaml is the deployment registry for the platform
  # documents. Discover manifests recursively so a new directory cannot escape
  # coverage. The Kustomization itself and OCI package metadata are deliberately
  # excluded: neither is a Kubernetes resource to install through this base.
  kustomization = YAML.safe_load(File.read("platform/kustomization.yaml"))
  listed = kustomization.fetch("resources", []) || []
  excluded = ["platform/kustomization.yaml", "platform/function/package/crossplane.yaml"]
  expected = (Dir.glob("platform/**/*.{yaml,yml}") - excluded).
    map { |path| path.delete_prefix("platform/") }.sort
  missing = expected - listed
  extra = listed - expected
  abort "platform/kustomization.yaml: missing resources: #{missing.join(", ")}" unless missing.empty?
  abort "platform/kustomization.yaml: unexpected resources: #{extra.join(", ")}" unless extra.empty?

  # A composite renderer is the implementation of an XRD composite kind.
  # Both sets are declared in this repository, so a missing implementation or
  # a renderer that has no API definition is always a repository defect.
  xrd_kinds = Dir.glob("platform/apis/*.{yaml,yml}").flat_map do |path|
    YAML.load_stream(File.read(path)).compact.select do |document|
      document["kind"] == "CompositeResourceDefinition"
    end.map { |document| document.dig("spec", "names", "kind") }
  end.sort
  renderer_kinds = File.read("platform/function/fn.go").scan(/^\s*"([^"]+)":\s*\{/).flatten.sort
  missing = xrd_kinds - renderer_kinds
  extra = renderer_kinds - xrd_kinds
  abort "platform/function/fn.go: compositeRenderers missing XRD kinds: #{missing.join(", ")}" unless missing.empty?
  abort "platform/function/fn.go: compositeRenderers have no XRD: #{extra.join(", ")}" unless extra.empty?

  # Check the explicitly declared inventory plurals here. Go registry coverage
  # tests additionally resolve every renderer GVK against the pinned CRD map.
  inventory_plurals = File.read("platform/function/inventory.go").scan(/^\s*plural:\s*"([^"]+)"/).flatten
  activation_policy = YAML.load_stream(File.read("platform/provider/provider-grafana.yaml")).compact.find do |document|
    document["kind"] == "ManagedResourceActivationPolicy"
  end
  abort "platform/provider/provider-grafana.yaml: missing ManagedResourceActivationPolicy" if activation_policy.nil?
  activated = activation_policy.dig("spec", "activate") || []
  missing = inventory_plurals - activated
  abort "platform/provider/provider-grafana.yaml: ManagedResourceActivationPolicy missing inventory resources: #{missing.join(", ")}" unless missing.empty?

  # Walk every XRD version schema and enforce the structural constraints for
  # Kubernetes set and map lists. These are decidable from the documents, so
  # this check does not need an API server or envtest assets.
  schema_walk = nil
  schema_walk = lambda do |value, path|
    case value
    when Hash
      if value.key?("x-kubernetes-list-type")
        list_type = value["x-kubernetes-list-type"]
        items = value["items"]
        case list_type
        when "set"
          if items.is_a?(Hash) && items["type"] == "object" && items["x-kubernetes-map-type"] != "atomic"
            abort "#{path}: x-kubernetes-list-type=set with object items requires items.x-kubernetes-map-type=atomic"
          end
        when "map"
          keys = value["x-kubernetes-list-map-keys"]
          abort "#{path}: x-kubernetes-list-type=map requires a non-empty x-kubernetes-list-map-keys" unless keys.is_a?(Array) && !keys.empty?
          required = items.is_a?(Hash) ? items["required"] : nil
          keys.each do |key|
            abort "#{path}: x-kubernetes-list-type=map key #{key} must appear in items.required" unless required.is_a?(Array) && required.include?(key)
          end
        end
      end
      value.each { |key, child| schema_walk.call(child, "#{path}.#{key}") }
    when Array
      value.each_with_index { |child, index| schema_walk.call(child, "#{path}[#{index}]") }
    end
  end

  Dir.glob("platform/apis/*.{yaml,yml}").sort.each do |path|
    YAML.load_stream(File.read(path)).each_with_index do |document, document_index|
      next unless document.is_a?(Hash)
      versions = document.dig("spec", "versions")
      next unless versions.is_a?(Array)

      versions.each_with_index do |version, version_index|
        next unless version.is_a?(Hash)
        schema = version.dig("schema", "openAPIV3Schema")
        next unless schema.is_a?(Hash)
        schema_walk.call(schema, "#{path} document #{document_index + 1} spec.versions[#{version_index}].schema.openAPIV3Schema")
      end
    end
  end

  # Installation manifests are self-contained signed-package pairs. Discover
  # every package resource and verification Job instead of maintaining the two
  # current filenames by hand, then require the exact digest to agree.
  Dir.glob("platform/**/*.{yaml,yml}").sort.each do |path|
    documents = YAML.load_stream(File.read(path)).compact
    packages = documents.select do |document|
      %w[Function Provider].include?(document["kind"]) && document.dig("spec", "package").is_a?(String)
    end
    verification_jobs = documents.select do |document|
      args = document.dig("spec", "template", "spec", "containers", 0, "args") || []
      document["kind"] == "Job" && args.include?("verify") && args.any? { |argument| argument.include?("@sha256:") }
    end
    if packages.empty?
      abort "#{path}: verification Job has no package resource" unless verification_jobs.empty?
      next
    end
    abort "#{path}: package resource has no verification Job" if verification_jobs.empty?
    abort "#{path}: expected one package resource and one verification Job" unless packages.length == 1 && verification_jobs.length == 1

    installed = packages.fetch(0).dig("spec", "package")
    verified = verification_jobs.fetch(0).dig("spec", "template", "spec", "containers", 0, "args").grep(/@sha256:/)
    abort "#{path}: package resource has no digest" unless installed.include?("@sha256:")
    abort "#{path}: verification Job must carry exactly one digest argument" unless verified.length == 1
    abort "#{path}: verifies #{verified.fetch(0)} but installs #{installed}" unless verified.fetch(0) == installed
  end
'

if [[ -n $(gofmt -l platform/function/*.go) ]]; then
  echo "Go source is not formatted:" >&2
  gofmt -l platform/function/*.go >&2
  exit 1
fi

(
  cd platform/function
  go mod tidy
  git diff --exit-code -- go.mod go.sum
  go test -v -race -cover ./...
  go vet ./...
)

find . -type f \( -name '*.yaml' -o -name '*.yml' \) -not -path './.git/*' -print0 |
  xargs -0 ruby -ryaml -e 'ARGV.each { |path| YAML.parse_stream(File.read(path)) }'

kubectl kustomize platform >/dev/null
kubectl kustomize deploy/aws >/dev/null

test -f examples/README.md

# Every catalog directory is enumerated, never listed by hand. A consumer applying
# Kustomize patches forces a render of every selected catalog path, so a directory
# without a kustomization.yaml fails before deployment even though its documents
# parse in isolation. A hand-maintained list silently stops covering new directories.
shopt -s nullglob
catalog_dirs=(examples/catalog/*/)
shopt -u nullglob
if [[ ${#catalog_dirs[@]} -eq 0 ]]; then
  echo "No catalog examples found under examples/catalog" >&2
  exit 1
fi
for example_dir in "${catalog_dirs[@]}"; do
  example_dir=${example_dir%/}
  if [[ ! -f "$example_dir/README.md" ]]; then
    echo "Missing example README: $example_dir/README.md" >&2
    exit 1
  fi
  if [[ ! -f "$example_dir/kustomization.yaml" ]]; then
    echo "Missing example kustomization: $example_dir/kustomization.yaml" >&2
    exit 1
  fi
  ruby -ryaml -e '
    kustomization_path, example_dir = ARGV
    listed = YAML.safe_load(File.read(kustomization_path)).fetch("resources", []) || []
    expected = Dir.glob(File.join(example_dir, "*.{yaml,yml}")).reject { |path| File.basename(path) == "kustomization.yaml" }.map { |path| File.basename(path) }.sort
    missing = expected - listed
    extra = listed - expected
    abort "#{kustomization_path}: missing resources: #{missing.join(", ")}" unless missing.empty?
    abort "#{kustomization_path}: unexpected resources: #{extra.join(", ")}" unless extra.empty?
  ' "$example_dir/kustomization.yaml" "$example_dir"
  kubectl kustomize "$example_dir" >/dev/null
done

ruby -ryaml -e '
  document = YAML.safe_load(File.read("deploy/argocd/requests-applicationset.yaml"))
  directories = document.dig("spec", "generators", 0, "git", "directories")
  abort "ApplicationSet must watch only top-level enabled/*" unless directories == [{"path" => "enabled/*"}]
'

echo "Validation passed."
