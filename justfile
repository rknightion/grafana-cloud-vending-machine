set shell := ["bash", "-euo", "pipefail", "-c"]

# renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools extractVersion=^envtest-v(?<version>.*)$
export ENVTEST_KUBERNETES_VERSION := "1.37.0"

# show the task surface
default:
    @just --list

# verify the local toolchain the repo's recipes assume is present
setup:
    @command -v rg >/dev/null
    @command -v ruby >/dev/null
    @command -v kubectl >/dev/null
    @command -v go >/dev/null
    @test -x "${KUBEBUILDER_ASSETS:?Run just envtest, then export the KUBEBUILDER_ASSETS path it prints}/kube-apiserver" && test -x "${KUBEBUILDER_ASSETS}/etcd" || { echo 'KUBEBUILDER_ASSETS has no kube-apiserver and etcd. Run just envtest and export the path it prints.' >&2; exit 1; }
    @actual_version="$("${KUBEBUILDER_ASSETS}/kube-apiserver" --version)"; expected_version="Kubernetes v${ENVTEST_KUBERNETES_VERSION}"; test "$actual_version" = "$expected_version" || { echo "Expected envtest $expected_version, found $actual_version" >&2; exit 1; }
    cd platform/function && go mod download

# format Go source and the justfile in place
[group('check')]
fmt:
    cd platform/function && gofmt -l -s -w .
    just --fmt

# verify formatting without mutating; never modifies files
[group('check')]
[no-exit-message]
fmt-check:
    cd platform/function && test -z "$(gofmt -l .)" || (cd platform/function && gofmt -l . && exit 1)
    just --fmt --check

# go vet plus the go.mod/go.sum tidiness check CI enforces
[group('check')]
[no-exit-message]
lint:
    cd platform/function && go vet ./...
    cd platform/function && go mod tidy
    cd platform/function && git diff --exit-code -- go.mod go.sum

# race-enabled Go test suite with coverage; optional filter=<pattern> narrows by -run
[group('check')]
[no-exit-message]
test filter="":
    cd platform/function && go test -race -cover {{ if filter != "" { "-run " + filter } else { "" } }} ./...

# scan the working tree and reachable Git history for source-environment identifiers
[group('check')]
public-release-scan:
    ./scripts/public-release-scan.sh

# THE GATE — exactly what CI runs; byte-for-byte the same script, zero drift
[group('check')]
[no-exit-message]
check:
    ./scripts/validate.sh

# build the composition function binary for the host platform
[group('build')]
build:
    cd platform/function && go build -trimpath -o bin/function-server .

# build the function's runtime container image locally (no push)
[group('build')]
image tag="function-grafana-vending:dev":
    docker buildx build --platform linux/amd64 --target image -t {{ tag }} platform/function

# download and verify the pinned envtest API-server assets the admission gate needs
[group('dev')]
envtest:
    #!/usr/bin/env bash
    set -euo pipefail
    version="${ENVTEST_KUBERNETES_VERSION}"
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$(uname -m)" in
      x86_64 | amd64) arch=amd64 ;;
      arm64 | aarch64) arch=arm64 ;;
      *) echo "No published envtest asset for machine type $(uname -m)" >&2; exit 1 ;;
    esac

    # Outside the repository tree deliberately: the gate walks every YAML document
    # under the working directory and the publication scan walks it for forbidden
    # filenames, so vendored third-party binaries have no business inside it.
    cache_root="${XDG_CACHE_HOME:-${HOME}/.cache}/grafana-cloud-vending-machine/envtest"

    # XDG_CACHE_HOME is caller-controlled, so prove the resolved location really
    # is outside the tree before writing 50 MB of binaries into it. Resolve
    # symlinks on both sides: an inside-the-tree cache is invisible to git and
    # fully visible to the two walks above.
    probe="$cache_root"
    while [[ ! -d "$probe" && "$probe" != "/" ]]; do probe="$(dirname "$probe")"; done
    resolved_cache="$(cd "$probe" && pwd -P)"
    resolved_repo="$(cd "$(git rev-parse --show-toplevel)" && pwd -P)"
    if [[ "$resolved_cache" == "$resolved_repo" || "$resolved_cache" == "$resolved_repo"/* ]]; then
      echo "Refusing to provision envtest assets inside the repository at ${cache_root}." >&2
      echo "Set XDG_CACHE_HOME to a location outside ${resolved_repo}." >&2
      exit 1
    fi
    extract_path="${cache_root}/envtest-v${version}-${os}-${arch}"
    assets_path="${extract_path}/controller-tools/envtest"
    expected_kubernetes_version="Kubernetes v${version}"

    if [[ -x "${assets_path}/kube-apiserver" && -x "${assets_path}/etcd" ]] &&
       [[ "$("${assets_path}/kube-apiserver" --version)" == "$expected_kubernetes_version" ]]; then
      echo "Pinned envtest assets already present."
    else
      release_url="https://github.com/kubernetes-sigs/controller-tools/releases/download/envtest-v${version}"
      archive_name="envtest-v${version}-${os}-${arch}.tar.gz"
      download_dir="$(mktemp -d)"
      trap 'rm -rf "$download_dir"' EXIT
      archive_path="${download_dir}/${archive_name}"

      curl --fail --location --retry 3 --retry-all-errors --output "$archive_path" "${release_url}/${archive_name}"
      curl --fail --location --retry 3 --retry-all-errors --output "${archive_path}.sha512" "${release_url}/${archive_name}.sha512"

      # The published sidecar is a single `<hash>  /<asset>` line. Anything else is
      # a changed release convention, not a checksum to guess at.
      expected_sha512="$(awk -v asset="/${archive_name}" '
        NR == 1 && $1 ~ /^[[:xdigit:]]+$/ && length($1) == 128 && $2 == asset && NF == 2 { print $1; next }
        { exit 1 }
        END { if (NR != 1) exit 1 }
      ' "${archive_path}.sha512")" || { echo "Published checksum sidecar has an unexpected format" >&2; exit 1; }

      # sha512sum is GNU-only; shasum ships with macOS. Never transcribe the hash.
      if command -v sha512sum >/dev/null; then checksum_check=(sha512sum --check -); else checksum_check=(shasum -a 512 --check -); fi
      printf '%s  %s\n' "$expected_sha512" "$archive_name" | (cd "$download_dir" && "${checksum_check[@]}")
      echo "Verified ${archive_name} against ${release_url}/${archive_name}.sha512"

      rm -rf "$extract_path"
      mkdir -p "$extract_path"
      tar --extract --gzip --file "$archive_path" --directory "$extract_path"
    fi

    test -x "${assets_path}/kube-apiserver" || { echo "Extracted archive has no kube-apiserver at ${assets_path}" >&2; exit 1; }
    test -x "${assets_path}/etcd" || { echo "Extracted archive has no etcd at ${assets_path}" >&2; exit 1; }
    kube_apiserver_version="$("${assets_path}/kube-apiserver" --version)"
    test "$kube_apiserver_version" = "$expected_kubernetes_version" || { echo "Expected envtest ${expected_kubernetes_version}, found ${kube_apiserver_version}" >&2; exit 1; }

    echo "kube-apiserver: ${kube_apiserver_version}"
    echo "etcd: $("${assets_path}/etcd" --version | head -n 1)"
    echo
    echo "Export this before running the gate:"
    # %q so a cache path containing a space stays copy-pasteable.
    printf 'export KUBEBUILDER_ASSETS=%q\n' "$assets_path"
