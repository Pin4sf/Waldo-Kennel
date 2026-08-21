#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

node -e '
const [major, minor, patch] = process.versions.node.split(".").map(Number);
if (major !== 22 || minor < 23 || (minor === 23 && patch < 2)) {
  console.error(`Kennel requires Node 22.23.2 through 22.x; found ${process.versions.node}`);
  process.exit(1);
}
'

npm ci
npm --prefix packages/product-ui ci
npm --prefix packages/cloud-client ci
npm --prefix frontend/acp-runtime ci --ignore-scripts
npm --prefix frontend ci
npm --prefix frontend/src/landing ci
npm --prefix scripts ci

echo "Kennel foundation dependencies installed."
