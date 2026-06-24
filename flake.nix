{
  description = "terraform-provider-kanidm development shell";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Pin Go version to match go.mod.
        go = pkgs.go_1_23;

        # Tools used during development and CI.
        devTools = with pkgs; [
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
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = devTools;

          # Environment variables
          env = {
            # Ensure CGO is disabled for reproducible cross-compilation.
            CGO_ENABLED = "0";
            # Point GOPATH to a project-local directory to keep installs isolated.
            GOPATH = toString ./. + "/.gopath";
          };

          shellHook = ''
            echo "🔧 terraform-provider-kanidm dev shell"
            echo "   Go:       $(go version)"
            echo "   OpenTofu: $(tofu version 2>/dev/null | head -1 || echo 'not found')"
            echo ""

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

        # Expose a formatter for `nix fmt`.
        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
