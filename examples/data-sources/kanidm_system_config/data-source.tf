data "kanidm_system_config" "this" {}

output "kanidm_domain" {
  value = data.kanidm_system_config.this.domain
}

output "ldap_enabled" {
  value = data.kanidm_system_config.this.ldap_enabled
}
