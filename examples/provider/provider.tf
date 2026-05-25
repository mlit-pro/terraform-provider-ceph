provider "ceph" {
  endpoint = "https://ceph-mgr.example.local:8443"
  username = "terraform"
  password = var.ceph_password # sensitive

  # Trust a private CA (alternative to insecure = true):
  # ca_cert = file("${path.module}/ceph-ca.pem")

  # For dev/test only:
  # insecure = true
}

# All five attributes can also be supplied via environment variables:
#   CEPH_ENDPOINT, CEPH_USERNAME, CEPH_PASSWORD, CEPH_CA_CERT, CEPH_INSECURE
