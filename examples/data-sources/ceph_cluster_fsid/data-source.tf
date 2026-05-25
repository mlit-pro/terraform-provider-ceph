data "ceph_cluster_fsid" "this" {}

output "ceph_cluster_fsid" {
  value = data.ceph_cluster_fsid.this.fsid
}
