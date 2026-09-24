#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

# Fail here rather than four minutes in. Without these the admission harness
# cannot start an API server, so every admission test fails individually, the
# suite still reports the rest as passing, and coverage silently drops - which
# reads as a broken repository instead of a missing prerequisite. Hosted CI
# provisions the assets and exports the path before calling this script.
if [[ ! -x "${KUBEBUILDER_ASSETS:-}/kube-apiserver" || ! -x "${KUBEBUILDER_ASSETS:-}/etcd" ]]; then
  echo "The admission gate needs the pinned envtest assets." >&2
  echo "Run 'just envtest', then export the KUBEBUILDER_ASSETS path it prints." >&2
  exit 1
fi

./scripts/public-release-scan.sh
./scripts/refused-shapes.sh

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

  # Derive the Kubernetes floor from every API object shipped by the platform
  # base. Dependency-provided and Kustomize API versions have no independent
  # Kubernetes floor here; every built-in object/kind pair must have an explicit
  # floor so a newly introduced cluster API cannot silently escape this check.
  dependency_api_versions = [
    "apiextensions.crossplane.io/v1",
    "apiextensions.crossplane.io/v1alpha1",
    "apiextensions.crossplane.io/v2",
    "meta.pkg.crossplane.io/v1",
    "pkg.crossplane.io/v1",
    "pkg.crossplane.io/v1beta1",
  ]
  non_resource_api_versions = ["kustomize.config.k8s.io/v1beta1"]
  built_in_floors = {
    ["admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicy"] => "1.30",
    ["admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicyBinding"] => "1.30",
    ["batch/v1", "Job"] => "1.21",
    ["rbac.authorization.k8s.io/v1", "ClusterRole"] => "1.8",
  }
  discovered_api_objects = Dir.glob("platform/**/*.{yaml,yml}").sort.flat_map do |path|
    YAML.load_stream(File.read(path)).compact.map do |document|
      next unless document.is_a?(Hash)
      api_version = document["apiVersion"]
      kind = document["kind"]
      next if api_version.nil? || kind.nil?
      [api_version, kind, path]
    end.compact
  end
  unknown_api_objects = discovered_api_objects.reject do |api_version, kind, _path|
    dependency_api_versions.include?(api_version) ||
      non_resource_api_versions.include?(api_version) ||
      built_in_floors.key?([api_version, kind])
  end
  unless unknown_api_objects.empty?
    details = unknown_api_objects.map { |api_version, kind, path| "#{path}: #{api_version} #{kind}" }.uniq.sort
    abort "platform manifests use API objects with no Kubernetes-floor classification: #{details.join(", ")}"
  end

  required_floors = discovered_api_objects.map do |api_version, kind, _path|
    built_in_floors[[api_version, kind]]
  end.compact
  abort "platform manifests: no built-in Kubernetes API requirement found" if required_floors.empty?
  version_key = lambda { |version| version.split(".").map(&:to_i) }
  required_kubernetes_floor = required_floors.max_by { |version| version_key.call(version) }

  installation_path = "docs/installation.md"
  kubernetes_rows = File.readlines(installation_path).select do |line|
    line.match?(/^\|\s*Kubernetes\s*\|/)
  end
  abort "#{installation_path}: pinned-versions table must contain exactly one Kubernetes row" unless kubernetes_rows.length == 1
  documented_floor = kubernetes_rows.fetch(0).split("|").fetch(2, "").match(/\b(\d+\.\d+)\b/)
  abort "#{installation_path}: Kubernetes row has no major.minor floor" if documented_floor.nil?
  documented_kubernetes_floor = documented_floor[1]
  unless documented_kubernetes_floor == required_kubernetes_floor
    abort "#{installation_path}: documents Kubernetes floor #{documented_kubernetes_floor}, platform manifests require #{required_kubernetes_floor}"
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
    abort "#{path}: package resource has no digest" unless installed.include?("@sha256:")
    package_digest = installed.split("@sha256:", 2).fetch(1)
    verification_job = verification_jobs.fetch(0)
    job_name = verification_job.dig("metadata", "name")
    abort "#{path}: verification Job must have a stable name" unless job_name.is_a?(String) && !job_name.empty?
    digest_prefixes = (8..package_digest.length).map { |length| package_digest[0, length] }
    abort "#{path}: verification Job name must not contain a digest or end with the package digest or a digest prefix" if job_name.match?(/sha256|[0-9a-f]{12,}/) || digest_prefixes.any? { |prefix| job_name.end_with?(prefix) }
    annotations = verification_job.dig("metadata", "annotations") || {}
    abort "#{path}: verification Job must be a PreSync hook" unless annotations["argocd.argoproj.io/hook"] == "PreSync"
    delete_policy = annotations["argocd.argoproj.io/hook-delete-policy"].to_s.split(",")
    abort "#{path}: verification Job must delete its predecessor before creation" unless delete_policy.include?("BeforeHookCreation")

    if packages.fetch(0)["kind"] == "Function"
      documentation = File.read(installation_path)
      documented_job = documentation.match(/The supplied install manifest\x27s verification Job is named `([^`]+)`;/)
      abort "#{installation_path}: must state the supplied Function verification Job name" if documented_job.nil?
      abort "#{installation_path}: supplied Function verification Job is #{documented_job[1]}, manifest has #{job_name}" unless documented_job[1] == job_name
    end

    verified = verification_job.dig("spec", "template", "spec", "containers", 0, "args").grep(/@sha256:/)
    abort "#{path}: verification Job must carry exactly one digest argument" unless verified.length == 1
    abort "#{path}: verifies #{verified.fetch(0)} but installs #{installed}" unless verified.fetch(0) == installed

    package_references = []
    collect_package_references = nil
    collect_package_references = lambda do |value|
      case value
      when Hash
        value.each_value { |child| collect_package_references.call(child) }
      when Array
        value.each { |child| collect_package_references.call(child) }
      when String
        package_references << value if value.include?(installed) || value.include?(package_digest)
      end
    end
    documents.each { |document| collect_package_references.call(document) }
    abort "#{path}: package digest must occur only in the package resource and verification Job argument" unless package_references.length == 2
  end

  # Documentation quotes pinned digests and names platform/ as canonical for
  # them. A digest that reaches a doc and is then superseded by a publish is
  # invisible to every other check here, so discover both sides and require
  # every documented digest to still exist in a platform manifest.
  digest_pattern = /sha256:[0-9a-f]{64}/
  platform_digests = Dir.glob("platform/**/*.{yaml,yml,json}").sort.flat_map do |path|
    File.read(path).scan(digest_pattern)
  end.uniq
  Dir.glob("docs/**/*.md").sort.each do |path|
    File.read(path).scan(digest_pattern).uniq.each do |digest|
      abort "#{path}: documents #{digest}, which no platform manifest pins" unless platform_digests.include?(digest)
    end
  end

  # A pinned-versions table is a public restatement of machine-readable
  # manifests and module requirements. Each row names its authoritative source
  # path, so new rows register themselves without a validator edit.
  markdown_paths = (Dir.glob("docs/**/*.md") + Dir.glob("examples/**/*.md") + ["README.md"]).
    select { |path| File.file?(path) }.uniq.sort
  pinned_table_paths = markdown_paths.select do |path|
    File.read(path).include?("| Component | Version | Source | Why |")
  end
  abort "documentation: no pinned-versions table found" if pinned_table_paths.empty?
  pinned_table_paths.each do |path|
    lines = File.readlines(path)
    header_index = lines.index { |line| line.include?("| Component | Version | Source | Why |") }
    index = header_index + 2
    while index < lines.length && lines.fetch(index).lstrip.start_with?("|")
      cells = lines.fetch(index).split("|").map(&:strip)
      component = cells.fetch(1, "")
      version_cell = cells.fetch(2, "")
      source_tokens = cells.fetch(3, "").scan(/`([^`]+)`/).flatten
      sources, locators = source_tokens.partition do |token|
        token.start_with?("deploy/", "platform/") && !token.include?(" ")
      end
      abort "#{path}: pinned component #{component} names no source path" if sources.empty?
      sources.each do |source|
        unless source.start_with?("deploy/", "platform/") && File.file?(source)
          abort "#{path}: pinned component #{component} source is missing or outside deploy/ and platform/: #{source}"
        end
      end
      abort "#{path}: pinned component #{component} names no exact source locator" if locators.empty?
      scalar_values = lambda do |value|
        case value
        when Hash
          value.flat_map { |key, child| [key.to_s] + scalar_values.call(child) }
        when Array
          value.flat_map { |child| scalar_values.call(child) }
        else
          [value.to_s]
        end
      end
      values_for_key = nil
      values_for_key = lambda do |value, key|
        case value
        when Hash
          own = value.key?(key) ? scalar_values.call(value.fetch(key)) : []
          own + value.values.flat_map { |child| values_for_key.call(child, key) }
        when Array
          value.flat_map { |child| values_for_key.call(child, key) }
        else
          []
        end
      end
      resolved_locator_values = lambda do |source, locator|
        if source.end_with?(".yaml", ".yml", ".json")
          documents = YAML.load_stream(File.read(source)).compact
          if locator.match?(/\A[A-Za-z0-9_.-]+:\z/)
            key = locator.delete_suffix(":")
            documents.flat_map { |document| values_for_key.call(document, key) }
          else
            documents.flat_map { |document| scalar_values.call(document) }.
              select { |value| value.include?(locator) }
          end
        else
          File.readlines(source).each_with_object([]) do |line, values|
            value = line.sub(%r{\s+//.*$}, "")
            values << value if value.include?(locator)
          end
        end
      end
      source_versions = sources.to_h do |source|
        resolved_values = locators.flat_map do |locator|
          resolved_locator_values.call(source, locator)
        end
        if resolved_values.empty?
          abort "#{path}: pinned component #{component} locator is absent from #{source}"
        end
        versions = resolved_values.flat_map do |value|
          value.scan(/\bv?(\d+\.\d+\.\d+)\b/).flatten
        end.uniq
        [source, versions]
      end
      version_cell.scan(/\bv?(\d+\.\d+\.\d+)\b/).flatten.each do |version|
        source_versions.each do |source, versions|
          unless versions.include?(version)
            abort "#{path}: documents #{component} version #{version}, #{source} locator pins #{versions.join(", ")}"
          end
        end
      end
      locator_text = locators.join("\n")
      version_cell.scan(digest_pattern).uniq.each do |digest|
        abort "#{path}: documents #{component} #{digest}, source locator does not pin it" unless locator_text.include?(digest)
      end
      index += 1
    end
  end

  # Documents that declare a complete XRD or catalog inventory must agree with
  # the discovered repository inventory. Select them by their section heading,
  # so a new inventory document is covered without adding its filename here.
  section_body = lambda do |text, heading|
    lines = text.lines
    start = lines.index { |line| line.strip == heading }
    next nil if start.nil?
    body = []
    lines[(start + 1)..-1].each do |line|
      break if line.start_with?("## ")
      body << line
    end
    body.join
  end

  xrd_inventory_paths = markdown_paths.select do |path|
    !section_body.call(File.read(path), "## CompositeResourceDefinitions").nil?
  end
  abort "documentation: no CompositeResourceDefinitions inventory found" if xrd_inventory_paths.empty?
  xrd_inventory_paths.each do |path|
    body = section_body.call(File.read(path), "## CompositeResourceDefinitions")
    documented = body.scan(/\bGrafana[A-Za-z0-9]+\b/).uniq.sort
    missing = xrd_kinds - documented
    extra = documented - xrd_kinds
    abort "#{path}: missing shipped XRD kinds: #{missing.join(", ")}" unless missing.empty?
    abort "#{path}: documents XRD kinds not shipped: #{extra.join(", ")}" unless extra.empty?
  end

  catalog_dirs = Dir.glob("examples/catalog/*/").map do |path|
    File.basename(path.delete_suffix("/"))
  end.sort
  catalog_inventory_paths = markdown_paths.select do |path|
    text = File.read(path)
    !section_body.call(text, "## Catalog").nil? ||
      !section_body.call(text, "## Catalog directories").nil?
  end
  abort "documentation: no catalog inventory found" if catalog_inventory_paths.empty?
  catalog_inventory_paths.each do |path|
    text = File.read(path)
    body = section_body.call(text, "## Catalog") ||
      section_body.call(text, "## Catalog directories")
    documented = body.scan(%r{(?:examples/)?catalog/([a-z0-9-]+)/}).flatten.uniq.sort
    missing = catalog_dirs - documented
    extra = documented - catalog_dirs
    abort "#{path}: missing catalog directories: #{missing.join(", ")}" unless missing.empty?
    abort "#{path}: documents catalog directories not shipped: #{extra.join(", ")}" unless extra.empty?
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
  # No non-request ApplicationSet is permitted: every tracked or nonignored
  # untracked object is a live-request source and must use the one
  # inert-by-default input shape. Kubernetes List documents can embed objects,
  # so inspect their items recursively rather than treating the wrapper as a
  # non-ApplicationSet document.
  candidate_paths = [
    ["git", "ls-files", "-z", "--", "*.yaml", "*.yml", "*.json"],
    ["git", "ls-files", "-z", "--others", "--exclude-standard", "--", "*.yaml", "*.yml", "*.json"],
  ].flat_map { |command| IO.popen(command, "rb", &:read).split("\0") }.reject(&:empty?).uniq.sort
  argo_objects_in = nil
  argo_objects_in = lambda do |document|
    next [] unless document.is_a?(Hash)
    if %w[Application ApplicationSet].include?(document["kind"])
      [[document["kind"], document]]
    elsif document["kind"] == "List"
      (document["items"] || []).flat_map { |item| argo_objects_in.call(item) }
    else
      []
    end
  end
  argo_objects = candidate_paths.flat_map do |path|
    YAML.load_stream(File.read(path)).compact.flat_map do |document|
      argo_objects_in.call(document).map do |kind, object|
        [kind, path, object.dig("metadata", "name") || "<unnamed>", object]
      end
    end
  end
  application_sets = argo_objects.select do |kind, _path, _name, _document|
    kind == "ApplicationSet"
  end.map do |_kind, path, name, document|
    [path, name, document]
  end
  abort "ApplicationSet: no tracked ApplicationSet found" if application_sets.empty?

  application_sets.each do |path, name, document|
    generators = document.dig("spec", "generators")
    unless generators.is_a?(Array) && generators.length == 1 && generators.fetch(0).is_a?(Hash) && generators.fetch(0).keys == ["git"]
      abort "#{path}: ApplicationSet #{name} must contain exactly one git generator"
    end
    git_generator = generators.fetch(0).fetch("git")
    unless git_generator.is_a?(Hash)
      abort "#{path}: ApplicationSet #{name} git generator must use the known directory-only shape"
    end
    allowed_fields = %w[directories repoURL revision]
    unsupported_fields = git_generator.keys - allowed_fields
    unless unsupported_fields.empty?
      abort "#{path}: ApplicationSet #{name} git generator has unsupported fields: #{unsupported_fields.sort.join(", ")}"
    end
    missing_fields = allowed_fields - git_generator.keys
    unless missing_fields.empty?
      abort "#{path}: ApplicationSet #{name} git generator is missing fields from the known shape: #{missing_fields.sort.join(", ")}"
    end
    directories = git_generator["directories"]
    unless directories == [{"path" => "enabled/*"}]
      abort "#{path}: ApplicationSet #{name} must watch only top-level enabled/*"
    end

    # ApplicationSet templates can turn otherwise inert examples into live
    # requests when they source a path under examples/ or enabled/. Keep the
    # only known request template: one go-template source at exactly the path
    # emitted by the enabled/* directory generator. Refuse multi-source and
    # template patches because either can add or rewrite a source.
    spec = document["spec"]
    live_path_risk = "a source path under examples/ or enabled/ can make an inert example live"
    unless spec["goTemplate"] == true
      abort "#{path}: ApplicationSet #{name} must set spec.goTemplate: true; #{live_path_risk}"
    end
    if spec.key?("templatePatch")
      abort "#{path}: ApplicationSet #{name} spec.templatePatch is unsupported because it can rewrite the source; #{live_path_risk}"
    end

    template = spec["template"]
    template_spec = template.is_a?(Hash) ? template["spec"] : nil
    unless template_spec.is_a?(Hash)
      abort "#{path}: ApplicationSet #{name} must declare spec.template.spec.source.path exactly as \"{{.path.path}}\"; #{live_path_risk}"
    end
    if template_spec.key?("sources")
      abort "#{path}: ApplicationSet #{name} spec.template.spec.sources is unsupported; use only spec.template.spec.source.path exactly as \"{{.path.path}}\"; #{live_path_risk}"
    end

    template_source = template_spec["source"]
    unless template_source.is_a?(Hash)
      abort "#{path}: ApplicationSet #{name} spec.template.spec.source must be a mapping with path exactly \"{{.path.path}}\"; #{live_path_risk}"
    end
    chart = template_source["chart"]
    if !chart.nil? && chart != ""
      abort "#{path}: ApplicationSet #{name} template source chart is unsupported even when the path matches; only a directory path is allowed; #{live_path_risk}"
    end
    source_path = template_source["path"]
    if source_path.nil? && %w[chart ref].any? { |field| template_source[field].is_a?(String) && !template_source[field].empty? }
      abort "#{path}: ApplicationSet #{name} template source cannot be a pathless chart or ref; #{live_path_risk}"
    end
    unless source_path.is_a?(String) && source_path == "{{.path.path}}"
      abort "#{path}: ApplicationSet #{name} spec.template.spec.source.path must be exactly \"{{.path.path}}\"; #{live_path_risk}"
    end
  end

  applications = argo_objects.select do |kind, _path, _name, _document|
    kind == "Application"
  end.map do |_kind, path, name, document|
    [path, name, document]
  end
  abort "Application: no tracked Application found" if applications.empty?

  # Owner decision: fail closed on source paths outside this allow-list.
  # A path under examples/ or enabled/ can make an inert example live.
  allowed_application_source_paths = %w[deploy/aws platform]
  applications.each do |path, name, document|
    spec = document["spec"]
    abort "#{path}: Application #{name} must declare spec.source or spec.sources" unless spec.is_a?(Hash)

    # Inspect both fields so a disallowed path cannot hide in either declaration.
    sources = []
    single_source = spec["source"]
    sources << single_source unless single_source.nil?
    multiple_sources = spec["sources"]
    unless multiple_sources.nil?
      unless multiple_sources.is_a?(Array) && multiple_sources.all? { |source| source.is_a?(Hash) }
        abort "#{path}: Application #{name} spec.sources entries must be mappings"
      end
      sources.concat(multiple_sources)
    end
    abort "#{path}: Application #{name} must declare at least one source" if sources.empty?

    sources.each do |source|
      unless source.is_a?(Hash)
        abort "#{path}: Application #{name} spec.source must be a mapping"
      end

      source_path = source["path"]
      if source_path.nil?
        chart_source = source["chart"].is_a?(String) && !source["chart"].empty?
        ref_source = source["ref"].is_a?(String) && !source["ref"].empty?
        next if chart_source || ref_source
        abort "#{path}: Application #{name} has a pathless source without a chart or ref"
      end
      unless source_path.is_a?(String) && !source_path.empty?
        abort "#{path}: Application #{name} source path must be a non-empty string"
      end
      if source_path.start_with?("/", "\\") || source_path.match?(/\A[A-Za-z]:[\\\/]/)
        abort "#{path}: Application #{name} source path #{source_path.inspect} must be repository-relative"
      end

      normalized_parts = []
      source_path.tr("\\", "/").split("/").each do |part|
        next if part.empty? || part == "."
        if part == ".."
          if normalized_parts.empty? || normalized_parts.last == ".."
            normalized_parts << part
          else
            normalized_parts.pop
          end
        else
          normalized_parts << part
        end
      end
      normalized_path = normalized_parts.empty? ? "." : normalized_parts.join("/")
      unless allowed_application_source_paths.include?(normalized_path)
        abort "#{path}: Application #{name} source path #{source_path.inspect} normalizes to #{normalized_path.inspect}, which is outside the allow-list; sourcing examples/ or enabled/ can make an inert example live"
      end
    end
  end
'

echo "Validation passed."
