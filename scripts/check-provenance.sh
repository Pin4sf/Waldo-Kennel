#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

required_strings=(
	'"upstream": "https://github.com/Untrivial-ai/agent-orchestrator.git"'
	'"sourceSha": "66ba38b0fc16c65367a009148fee0cd5afb81a00"'
	'"sourceTree": "d06dd98b8f503b241e9f322ea2353b82442413e3"'
	'"license": "Apache-2.0"'
	'Agent Orchestrator'
)

for required in "${required_strings[@]}"; do
	if ! grep -Fq "$required" upstream.json NOTICE docs/upstream-provenance.md; then
		echo "check-provenance: missing required provenance text: $required" >&2
		exit 1
	fi
done

if ! grep -Fq 'Apache License' LICENSE; then
	echo "check-provenance: root LICENSE is not Apache-2.0" >&2
	exit 1
fi

echo "Provenance metadata is internally consistent."
