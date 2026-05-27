resource "ceph_user" "ci" {
  username = "ci"
  password = "change-me-please"
  roles    = ["read-only"]
  name     = "CI Bot"
  email    = "ci@example.com"
}
