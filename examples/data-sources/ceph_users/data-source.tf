data "ceph_users" "all" {}

output "usernames" {
  value = [for u in data.ceph_users.all.users : u.username]
}
