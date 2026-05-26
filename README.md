# Terraform Provider for Ceph

A Terraform / OpenTofu provider for managing a [Ceph](https://ceph.io) cluster through the
[Ceph Manager Dashboard REST API](https://docs.ceph.com/en/latest/mgr/ceph_api/).

## Disclaimer

This project is a personal open-source initiative and is not affiliated with, endorsed by, or
associated with any of the maintainer's current or former employers. All opinions, code, and
documentation are solely those of the maintainer and the individual contributors.

The project is not affiliated with the [Ceph project](https://ceph.io) or the
[Ceph Foundation](https://ceph.io/en/foundation/). The use of the Ceph name and/or logo is for
informational purposes only and does not imply any endorsement or affiliation with the Ceph project.

## Compatibility Promise

This provider targets the actively maintained Ceph releases: **Squid** (v19) and **Tentacle** (v20).
These are the only releases we test against and support.

> [!IMPORTANT]
> Older Ceph releases (Reef and earlier) are not supported. While some functionality might work, we
> do not test against them, and issues specific to those releases will not be addressed.

While the provider is on version 0.x, it is not guaranteed to be backward compatible with all
previous minor versions. However, we will try to maintain backward compatibility between provider
versions as much as possible.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.6
- [Go](https://golang.org/doc/install) >= 1.25

## Building

```shell
make build      # gofmt + go build -v ./...
make install    # installs the provider binary into $GOBIN
make generate   # regenerates docs/ from code via tfplugindocs
make test       # unit tests
make testacc    # acceptance tests (requires TF_ACC=1 and a reachable Ceph dashboard)
```

The provider address used in `required_providers` and during local development is
`registry.terraform.io/mlit-pro/ceph`.

## Using the Provider (local development)

After `make install`, point Terraform at the local plugin directory:

```hcl
terraform {
  required_providers {
    ceph = {
      source = "mlit-pro/ceph"
    }
  }
}

provider "ceph" {
  endpoint = "https://ceph-mgr.example.local:8443"
}
```

## License

Mozilla Public License Version 2.0 - see [LICENSE](./LICENSE).
