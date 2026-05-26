resource "ceph_cluster_user" "rbd" {
  entity = "client.rbd"
  capabilities = {
    mon = "profile rbd"
    osd = "profile rbd pool=rbd"
  }
}

output "rbd_keyring" {
  value     = ceph_cluster_user.rbd.keyring
  sensitive = true
}
