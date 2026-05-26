data "ceph_cluster_users" "all" {}

output "cluster_user_entities" {
  value = [for u in data.ceph_cluster_users.all.users : u.entity]
}
