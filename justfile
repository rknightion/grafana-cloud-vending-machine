set shell := ["bash", "-euo", "pipefail", "-c"]

# show the task surface
default:
    @just --list

# verify the local toolchain the repo's recipes assume is present
setup:
    @command -v rg >/dev/null
    @command -v ruby >/dev/null
    @command -v kubectl >/dev/null
    @command -v go >/dev/null
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
