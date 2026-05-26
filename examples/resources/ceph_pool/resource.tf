resource "ceph_pool" "rbd" {
  name                 = "rbd"
  pool_type            = "replicated"
  pg_num               = 32
  size                 = 3
  application_metadata = ["rbd"]
}

resource "ceph_pool" "ec_data" {
  name                 = "ec-data"
  pool_type            = "erasure"
  pg_num               = 32
  erasure_code_profile = "default"
  application_metadata = ["rgw"]
}
