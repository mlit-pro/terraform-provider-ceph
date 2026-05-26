resource "ceph_role" "pool_reader" {
  name        = "pool-reader"
  description = "Read-only access to pools and monitors"

  scopes_permissions = {
    pool    = ["read"]
    monitor = ["read"]
  }
}
