data "ceph_health" "this" {}

output "ceph_health_status" {
  value = data.ceph_health.this.health_status
}
