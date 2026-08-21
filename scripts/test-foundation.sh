#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

./scripts/check-provenance.sh

(
	cd backend
	go build ./...
	go test ./...
	go vet ./...
)

npm run shared:check
npm --prefix frontend run typecheck
npm --prefix frontend test
npm --prefix frontend/src/landing run build
node --test scripts/kennel-e2e-pod-gate.test.mjs

npm run sqlc
npm run api
git diff --exit-code -- \
	backend/internal/storage/sqlite/gen \
	backend/internal/httpd/apispec/openapi.yaml \
	frontend/src/api/schema.ts
