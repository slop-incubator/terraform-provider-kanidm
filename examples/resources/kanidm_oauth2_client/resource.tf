# Confidential (basic) client
resource "kanidm_oauth2_client" "myapp" {
  name         = "myapp"
  display_name = "My Application"
  origin       = "https://app.example.com"
  type         = "basic"

  redirect_uris = [
    "https://app.example.com/callback",
    "https://app.example.com/silent-callback",
  ]
}

# client_secret is sensitive and computed — access it like this:
output "myapp_client_secret" {
  value     = kanidm_oauth2_client.myapp.client_secret
  sensitive = true
}

# Public client with PKCE (e.g. a SPA or native app)
resource "kanidm_oauth2_client" "spa" {
  name         = "my-spa"
  display_name = "My Single-Page App"
  origin       = "https://spa.example.com"
  type         = "public"
  pkce_method  = "S256"

  redirect_uris = [
    "https://spa.example.com/callback",
  ]
}
