data "ceph_cluster_user" "admin" {
  entity = "client.admin"
}

output "admin_capabilities" {
  value = data.ceph_cluster_user.admin.capabilities
}
