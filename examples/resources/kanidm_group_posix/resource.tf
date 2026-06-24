resource "kanidm_group" "linux_users" {
  name = "linux-users"
}

resource "kanidm_group_posix" "linux_users" {
  group_id = kanidm_group.linux_users.id
  # gid_number is auto-allocated by Kanidm if omitted.
}
