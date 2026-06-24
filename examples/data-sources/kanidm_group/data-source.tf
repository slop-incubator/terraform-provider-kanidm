# Look up a group that is managed outside Terraform (e.g. a built-in system group).
data "kanidm_group" "idm_admins" {
  name = "idm_admins"
}

output "idm_admins_members" {
  value = data.kanidm_group.idm_admins.members
}
