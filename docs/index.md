---
page_title: "Kanidm Provider"
description: |-
  The Kanidm provider manages resources in a Kanidm identity management server.
---

# Kanidm Provider

The **Kanidm** provider allows you to manage resources in a [Kanidm](https://kanidm.github.io/kanidm/)
identity management server using OpenTofu or Terraform.

## Authentication

Create a dedicated service account scoped to only the permissions your automation needs.
Using the built-in `idm_admin` account is strongly discouraged.

```shell
# Create a service account for Terraform/OpenTofu
kanidm service-account create tofu-automation "OpenTofu Automation"

# Add it to the groups it needs to manage resources
kanidm group add-members idm_group_admins tofu-automation

# Generate an API token
kanidm service-account api-token generate tofu-automation tofu-provider --expiry 365d
```

Supply the token via the `KANIDM_TOKEN` environment variable:

```shell
export KANIDM_URL=https://idm.example.com
export KANIDM_TOKEN=<your-token>
```

## State Security

This provider writes sensitive values into Terraform state:

- `kanidm_oauth2_client.client_secret` — the OAuth2 client secret for `basic` clients.

**Always use an encrypted remote backend** (e.g. S3 + SSE-KMS, HCP Terraform, or Vault).
Never commit plain `.tfstate` files to source control.

## Provider Configuration

```hcl
terraform {
  required_providers {
    kanidm = {
      source  = "registry.opentofu.org/slop-incubator/kanidm"
      version = "~> 0.1"
    }
  }
}

provider "kanidm" {
  url   = "https://idm.example.com"  # or KANIDM_URL env var
  token = var.kanidm_token            # or KANIDM_TOKEN env var
}
```

## Schema

### Required

- `url` (String) — Base URL of the Kanidm instance. Can also be set via `KANIDM_URL`.
- `token` (String, Sensitive) — Bearer token. Can also be set via `KANIDM_TOKEN`.

### Optional

- `tls_skip_verify` (Boolean) — Disable TLS certificate verification. **Development only.**
  Set `KANIDM_TLS_STRICT=1` in CI to treat this as an error when targeting non-localhost URLs.
- `timeout` (String) — HTTP client timeout (Go duration, e.g. `"30s"`). Default: `"30s"`.
