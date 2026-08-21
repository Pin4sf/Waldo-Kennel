#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

for package_dir in . packages/product-ui packages/cloud-client frontend/acp-runtime frontend frontend/src/landing; do
	echo "Auditing production dependencies: $package_dir"
	npm --prefix "$package_dir" audit --omit=dev --audit-level=high
done
