data "kanidm_service_account" "existing_app" {
  spn = "existingapp@example.com"
}

output "existing_app_id" {
  value = data.kanidm_service_account.existing_app.id
}
