data "ceph_pool" "rbd" {
  name = "rbd"
}

output "rbd_pg_num" {
  value = data.ceph_pool.rbd.pg_num
}
