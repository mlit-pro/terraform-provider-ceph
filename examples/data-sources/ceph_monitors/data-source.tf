data "ceph_monitors" "all" {}

output "mon_addrs" {
  value = data.ceph_monitors.all.monitors[*].public_addr
}
