# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- This changelog, following the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format.
- CI enforcement requiring a changelog entry on pull requests.
- Release notes published from the changelog via GoReleaser.
- Provider configuration (`endpoint`, `username`, `password`, `ca_cert`, `insecure`) with
  `CEPH_*` environment-variable fallbacks, and a minimal Ceph Manager Dashboard API client
  handling TLS and JWT authentication.

### Changed

- CI now runs unit tests on every pull request; acceptance tests moved to a manually-dispatched
  `Acceptance Tests` workflow since they require a live Ceph cluster.

[unreleased]: https://github.com/mlit-pro/terraform-provider-ceph/commits/master
