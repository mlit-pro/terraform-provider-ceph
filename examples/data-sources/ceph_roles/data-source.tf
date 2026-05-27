data "ceph_roles" "all" {}

output "role_names" {
  value = [for r in data.ceph_roles.all.roles : r.name]
}
