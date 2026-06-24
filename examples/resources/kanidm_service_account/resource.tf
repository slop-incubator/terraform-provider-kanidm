resource "kanidm_service_account" "myapp" {
  name         = "myapp"
  display_name = "My Application"
}

# Service account managed by the platform team
resource "kanidm_service_account" "ci_runner" {
  name             = "ci-runner"
  display_name     = "CI Runner"
  entry_managed_by = kanidm_group.platform.spn
}
