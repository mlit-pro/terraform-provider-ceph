# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-26

### Added

- Provider configuration (`endpoint`, `username`, `password`, `ca_cert`, `insecure`) with
  `CEPH_*` environment-variable fallbacks, and a minimal Ceph Manager Dashboard API client
  handling TLS and JWT authentication.
- `ceph_health` data source exposing the cluster's overall health status.
- `ceph_cluster_fsid` data source exposing the cluster's FSID.
- `ceph_cluster_capacity` data source exposing the cluster's raw capacity
  (total, available, and used bytes).
- `ceph_cluster_user` resource for managing CephX cluster users (`ceph auth` entities)
  and their capabilities, exposing the secret `key` and full `keyring`.
- `ceph_cluster_user` and `ceph_cluster_users` data sources for reading CephX cluster users.
- `ceph_pool` resource for managing Ceph pools (replicated and erasure), including capabilities,
  quotas, autoscale mode, and in-place rename; create/update/delete wait for the async pool task.
- `ceph_pool` and `ceph_pools` data sources for reading Ceph pools.

[unreleased]: https://github.com/mlit-pro/terraform-provider-ceph/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/mlit-pro/terraform-provider-ceph/releases/tag/v0.1.0
