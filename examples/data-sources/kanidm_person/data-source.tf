data "kanidm_person" "admin" {
  spn = "admin@example.com"
}

output "admin_display_name" {
  value = data.kanidm_person.admin.display_name
}
