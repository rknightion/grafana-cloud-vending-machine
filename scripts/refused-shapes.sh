#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root/platform/function"

go test ./... -run '^TestRefusedVendorShapeAssertionCoversEveryEmittedKind$' -count=1
