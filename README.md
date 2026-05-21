# Terraform Provider for Ceph

A [Terraform](https://www.terraform.io) provider for managing a [Ceph](https://ceph.io) cluster
through the **Ceph Manager Dashboard REST API**
(`https://<host>:8443/api`, [docs](https://docs.ceph.com/en/reef/mgr/ceph_api/)).

> **Status:** scaffold. No real Ceph resources are wired up yet - this repo currently exposes the
> `ceph_example` placeholder resource, data source, function, action, and ephemeral resource that
> come from the [terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
> template. Real Ceph resources (pools, RGW users, RBD images, etc.) will land in follow-up changes.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
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
      source  = "mlit-pro/ceph"
      version = "0.0.1"
    }
  }
}

provider "ceph" {
  endpoint = "https://ceph-mgr.example.local:8443"
}
```

## License

Mozilla Public License Version 2.0 - see [LICENSE](./LICENSE).
