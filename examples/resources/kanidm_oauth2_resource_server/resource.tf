resource "kanidm_oauth2_resource_server" "myapi" {
  name         = "myapi"
  display_name = "My API"
  origin       = "https://api.example.com"

  scope_maps = {
    (kanidm_group.developers.spn) = ["openid", "profile", "email", "api:read"]
    (kanidm_group.ops.spn)        = ["openid", "profile", "email", "api:read", "api:write"]
  }
}
