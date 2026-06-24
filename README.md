# terraform-provider-kanidm

An [OpenTofu](https://opentofu.org) / [Terraform](https://terraform.io) provider for
[Kanidm](https://kanidm.github.io/kanidm/), backed by
[`go-kanidm`](https://github.com/slop-incubator/go-kanidm).

[![CI](https://github.com/slop-incubator/terraform-provider-kanidm/actions/workflows/ci.yml/badge.svg)](https://github.com/slop-incubator/terraform-provider-kanidm/actions/workflows/ci.yml)

---

## Managed Resources

| Resource | Description |
|---|---|
| `kanidm_person` | Person account |
| `kanidm_service_account` | Service account |
| `kanidm_group` | Group |
| `kanidm_person_posix` | POSIX extension for a person |
| `kanidm_group_posix` | POSIX extension for a group |
| `kanidm_oauth2_resource_server` | OAuth2 resource server (RS) |
| `kanidm_oauth2_client` | OAuth2 client (basic or public) |

## Data Sources

| Data Source | Description |
|---|---|
| `kanidm_person` | Read a person by SPN |
| `kanidm_group` | Read a group by name |
| `kanidm_service_account` | Read a service account by SPN |
| `kanidm_oauth2_client` | Read an OAuth2 client by name |
| `kanidm_system_config` | Read global system configuration |

---

## Quick Start

```hcl
terraform {
  required_providers {
    kanidm = {
      source  = "registry.opentofu.org/slop-incubator/kanidm"
      version = "~> 0.1"
    }
  }
}

provider "kanidm" {}  # configure via KANIDM_URL and KANIDM_TOKEN env vars

resource "kanidm_group" "developers" {
  name = "developers"
}

resource "kanidm_person" "alice" {
  name         = "alice"
  display_name = "Alice Example"
  mail         = ["alice@example.com"]
}
```

## Development

### Requirements

- [Nix](https://nixos.org/download) with flakes enabled, **or** Go 1.23+, golangci-lint, OpenTofu

### Using Nix (recommended)

```shell
nix develop          # enter the dev shell with all tools
make help            # list available targets
make build           # build the provider binary
make test            # run unit tests
```

### Without Nix

```shell
go build ./...
make test
```

### Acceptance Tests

Acceptance tests require a running Kanidm instance:

```shell
make test-acceptance   # starts Kanidm via Docker, runs full test suite
```

Or manually:

```shell
export KANIDM_URL=https://localhost:8443
export KANIDM_TOKEN=<token>
TF_ACC=1 go test -v ./...
```

### Scaffolding a New Resource

```shell
# 1. Create a spec YAML in tools/codegen/specs/
cp tools/codegen/specs/oauth2_client.yaml tools/codegen/specs/my_resource.yaml
# edit my_resource.yaml

# 2. Generate the scaffold
make scaffold RESOURCE=my_resource

# 3. Fill in the TODO markers in the generated files
```

### Checking for API Drift

```shell
make schema-diff KANIDM_URL=https://idm.example.com
```

---

## Security

- The `token` provider attribute is marked `Sensitive` and never appears in plan output.
- `kanidm_oauth2_client.client_secret` is sensitive and stored in state — use an encrypted backend.
- `tls_skip_verify = true` emits a warning on non-localhost URLs; set `KANIDM_TLS_STRICT=1` to treat it as an error in CI.
- Create a scoped service account for the provider rather than using `idm_admin`.

See [`docs/index.md`](docs/index.md) for full provider documentation.

---

## Contributing

Pull requests are welcome. Please:

1. Run `make lint` and `make test` before opening a PR.
2. Add acceptance tests for new resources.
3. Update `tools/schema-sync/baseline.json` when adding new attribute mappings.
4. Follow the security checklist in the implementation plan.

## License

[Mozilla Public License 2.0](LICENSE)
