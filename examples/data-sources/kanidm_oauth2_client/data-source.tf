# Note: client_secret is not available via this data source (the API does not return it on GET).
data "kanidm_oauth2_client" "existing_app" {
  name = "existing-app"
}

output "existing_app_origin" {
  value = data.kanidm_oauth2_client.existing_app.origin
}
