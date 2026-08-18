#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

upstream_url="$(sed -n 's/^[[:space:]]*"upstream": "\([^"]*\)",*$/\1/p' upstream.json)"
source_sha="$(sed -n 's/^[[:space:]]*"sourceSha": "\([0-9a-f]*\)",*$/\1/p' upstream.json)"
source_tree="$(sed -n 's/^[[:space:]]*"sourceTree": "\([0-9a-f]*\)",*$/\1/p' upstream.json)"

if [[ -z "$upstream_url" || -z "$source_sha" || -z "$source_tree" ]]; then
	echo "compare-upstream: upstream.json is incomplete" >&2
	exit 1
fi

if ! git cat-file -e "${source_sha}^{commit}" 2>/dev/null; then
	echo "Fetching recorded AO source commit ${source_sha}..." >&2
	git fetch --no-tags "$upstream_url" "$source_sha"
fi

actual_tree="$(git show -s --format=%T "$source_sha")"
if [[ "$actual_tree" != "$source_tree" ]]; then
	echo "compare-upstream: source tree mismatch: got ${actual_tree}, want ${source_tree}" >&2
	exit 1
fi

comparison_ref="${1:-HEAD}"
git rev-parse --verify "${comparison_ref}^{commit}" >/dev/null

shared_paths="$(comm -12 \
	<(git ls-tree -r --name-only "$source_sha" | LC_ALL=C sort) \
	<(git ls-tree -r --name-only "$comparison_ref" | LC_ALL=C sort) | wc -l | tr -d ' ')"
exact_blobs="$(comm -12 \
	<(git ls-tree -r --format='%(objectname) %(path)' "$source_sha" | LC_ALL=C sort) \
	<(git ls-tree -r --format='%(objectname) %(path)' "$comparison_ref" | LC_ALL=C sort) | wc -l | tr -d ' ')"

printf 'upstream=%s\nsource_sha=%s\nsource_tree=%s\ncomparison_ref=%s\nshared_paths=%s\nexact_same_path_blobs=%s\n' \
	"$upstream_url" "$source_sha" "$source_tree" "$comparison_ref" "$shared_paths" "$exact_blobs"
git diff --shortstat "$source_sha" "$comparison_ref" || true

