#!/usr/bin/env bash
# gen-docs.sh — Regenerate provider documentation using tfplugindocs.
# Requires tfplugindocs to be installed: go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

set -euo pipefail

echo "→ Running go generate..."
go generate ./...

echo "→ Generating provider docs..."
tfplugindocs generate --provider-name kanidm

echo "✓ Docs written to docs/"
