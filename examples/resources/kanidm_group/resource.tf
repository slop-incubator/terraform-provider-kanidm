# Minimal group
resource "kanidm_group" "developers" {
  name = "developers"
}

# Group with members (full-replace semantics on every apply)
resource "kanidm_group" "platform" {
  name    = "platform"
  members = [
    kanidm_person.alice.spn,
    kanidm_person.bob.spn,
  ]
}

# Group managed by another group
resource "kanidm_group" "ops" {
  name            = "ops"
  entry_managed_by = kanidm_group.platform.spn
}

# Group with POSIX attributes (for Linux login)
resource "kanidm_group" "linux_users" {
  name = "linux-users"
}

resource "kanidm_group_posix" "linux_users" {
  group_id = kanidm_group.linux_users.id
  # gid_number is auto-allocated by Kanidm if omitted
}
