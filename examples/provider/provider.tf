terraform {
  required_providers {
    kanidm = {
      source  = "registry.opentofu.org/slop-incubator/kanidm"
      version = "~> 0.1"
    }
  }
}

# Configure the provider via environment variables (recommended):
#   export KANIDM_URL=https://idm.example.com
#   export KANIDM_TOKEN=<your-token>
provider "kanidm" {}

# Or supply values explicitly (use variables — never hardcode secrets):
# provider "kanidm" {
#   url   = "https://idm.example.com"
#   token = var.kanidm_token
# }
