resource "kanidm_person" "alice" {
  name         = "alice"
  display_name = "Alice Example"
  mail         = ["alice@example.com"]
}

# Person with POSIX attributes (for Linux login / SSH)
resource "kanidm_person" "bob" {
  name         = "bob"
  display_name = "Bob Example"
}

resource "kanidm_person_posix" "bob" {
  person_id   = kanidm_person.bob.id
  login_shell = "/bin/bash"
  home_dir    = "/home/bob"
  # uid_number and gid_number are auto-allocated by Kanidm
}
