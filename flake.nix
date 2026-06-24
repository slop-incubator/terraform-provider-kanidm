{
  description = "terraform-provider-kanidm development shell";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          name = "terraform-provider-kanidm";

          packages = with pkgs; [
            # Go toolchain
            go
            gopls
            gotools        # goimports, godoc, etc.
            go-tools       # staticcheck
  
            # Linting
            golangci-lint
  
            # Terraform / OpenTofu ecosystem
            opentofu
            terraform-docs
  
            # Release tooling
            goreleaser
            gnupg
  
            # Documentation generation
            # tfplugindocs is installed via `go install` (see below) since nixpkgs
            # may lag behind the required version. The shell hook handles this.
  
            # Container tooling for acceptance tests
            docker-client
  
            # Utilities
            jq
            curl
            git
            gnumake
            yq-go          # YAML processing for schema-sync specs
          ];

          # Environment variables
          env = {
            # Ensure CGO is disabled for reproducible cross-compilation.
            CGO_ENABLED = "0";
          };

          shellHook = ''
            echo "🔧 terraform-provider-kanidm dev shell"
            echo "   Go:       $(go version)"
            echo "   OpenTofu: $(tofu version 2>/dev/null | head -1 || echo 'not found')"
            echo ""

            # ── Go environment ────────────────────────────────────────────
            # Keep the module download cache in the project root so it
            # persists across `nix develop` invocations and doesn't re-fetch
            # on every shell entry.
            export GOPATH="$PWD/.gopath"
            export GOMODCACHE="$GOPATH/pkg/mod"
            export GOCACHE="$GOPATH/cache/build"
            export PATH="$GOPATH/bin:$PATH"


            # Install tfplugindocs into the local GOPATH if not present.
            if ! command -v tfplugindocs &>/dev/null; then
              echo "→ Installing tfplugindocs..."
              go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
            fi

            # Add local GOPATH bin to PATH so `go install`-ed tools are available.
            export PATH="$GOPATH/bin:$PATH"

            echo "Run 'make help' to see available targets."
          '';
        };
      }
    );
}
