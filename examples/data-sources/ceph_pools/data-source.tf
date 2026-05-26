data "ceph_pools" "all" {}

output "pool_names" {
  value = [for p in data.ceph_pools.all.pools : p.name]
}
