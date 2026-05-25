data "ceph_cluster_capacity" "this" {}

output "ceph_cluster_capacity_bytes" {
  value = {
    total     = data.ceph_cluster_capacity.this.total_bytes
    available = data.ceph_cluster_capacity.this.total_avail_bytes
    used      = data.ceph_cluster_capacity.this.total_used_raw_bytes
  }
}
