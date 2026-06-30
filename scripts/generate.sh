#!/usr/bin/env bash
#
# generate.sh
#
# Generates Terraform provider code from an OpenAPI spec using:
#   1. hashicorp/terraform-plugin-codegen-openapi (tfplugingen-openapi)
#      -> turns openapi.json + generator_config.yml into a provider-code-spec.json
#   2. hashicorp/terraform-plugin-codegen-framework (tfplugingen-framework)
#      -> turns provider-code-spec.json into Go source for terraform-plugin-framework
#
# Usage:
#   ./generate.sh [openapi.json] [generator_config.yml] [output_dir]
#
# Defaults:
#   openapi.json         = ./openapi.json
#   generator_config.yml = ./generator_config.yml
#   output_dir            = ./generated

set -euo pipefail

# ---- Config / args ----------------------------------------------------

OPENAPI_SPEC="internal/spec/kanidm-openapi.json"
GENERATOR_CONFIG="internal/spec/generator_config.yml"
OUTPUT_DIR="internal/"

CODE_SPEC_FILE="internal/spec/provider-code-spec.json"
PACKAGE_NAME="kanidm"

# Pin versions if you want reproducible builds; "latest" is simplest to start.
OPENAPI_GEN_VERSION="${OPENAPI_GEN_VERSION:-latest}"
FRAMEWORK_GEN_VERSION="${FRAMEWORK_GEN_VERSION:-latest}"

# ---- Sanity checks ------------------------------------------------------

command -v go >/dev/null 2>&1 || { echo "Error: 'go' is required but not installed." >&2; exit 1; }

if [[ ! -f "$OPENAPI_SPEC" ]]; then
  echo "Error: OpenAPI spec not found at '$OPENAPI_SPEC'" >&2
  exit 1
fi

if [[ ! -f "$GENERATOR_CONFIG" ]]; then
  echo "Error: generator config not found at '$GENERATOR_CONFIG'" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR" "$OUTPUT_DIR/resources" "$OUTPUT_DIR/data_sources" "$OUTPUT_DIR/provider"

# ---- Step 0: ensure the generator binaries are installed ----------------

echo "==> Installing tfplugingen-openapi ($OPENAPI_GEN_VERSION)..."
go install "github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@${OPENAPI_GEN_VERSION}"

echo "==> Installing tfplugingen-framework ($FRAMEWORK_GEN_VERSION)..."
go install "github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@${FRAMEWORK_GEN_VERSION}"

# ---- Step 1: OpenAPI spec + generator config -> provider-code-spec.json -

echo "==> Generating provider code spec from OpenAPI..."
tfplugingen-openapi generate \
  --config "$GENERATOR_CONFIG" \
  --output "$CODE_SPEC_FILE" \
  "$OPENAPI_SPEC"

echo "    Wrote $CODE_SPEC_FILE"

# ---- Step 2: provider-code-spec.json -> Go provider code ----------------

echo "==> Generating Terraform provider Go code..."
tfplugingen-framework generate provider \
  --input "$CODE_SPEC_FILE" \
  --output "$OUTPUT_DIR/provider" \
  --package "$PACKAGE_NAME"

echo "==> Generating Terraform data_sources Go code..."
tfplugingen-framework generate data-sources \
  --input "$CODE_SPEC_FILE" \
  --output "$OUTPUT_DIR/data_sources" \
  --package "$PACKAGE_NAME"

echo "==> Generating Terraform resource Go code..."
tfplugingen-framework generate resources \
  --input "$CODE_SPEC_FILE" \
  --output "$OUTPUT_DIR/resources" \
  --package "$PACKAGE_NAME"


echo "==> Done. Generated provider code is in: $OUTPUT_DIR"
echo "    Next steps:"
echo "      - Review $OUTPUT_DIR for generated *_resource_gen.go / *_data_source_gen.go files"
echo "      - Wire up actual CRUD logic (these scaffolds are schema-only)"
echo "      - Run 'go build ./...' inside your provider module to confirm it compiles"