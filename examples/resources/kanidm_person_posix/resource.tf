resource "kanidm_person" "alice" {
  name         = "alice"
  display_name = "Alice Example"
}

resource "kanidm_person_posix" "alice" {
  person_id   = kanidm_person.alice.id
  login_shell = "/bin/bash"
  home_dir    = "/home/alice"
  # uid_number and gid_number are auto-allocated by Kanidm if omitted.
}
