# Terraform Provider Shireesh (Terraform Plugin Framework)

This repository contains a Terraform provider named `shireesh`.

- Provider address: `registry.terraform.io/gshireesh/shireesh`
- Module path: `github.com/gshireesh/terraform-provider-shireesh`

## Quickstart

```terraform
terraform {
  required_providers {
    shireesh = {
      source  = "gshireesh/shireesh"
      version = ">= 0.0.1"
    }
  }
}

provider "shireesh" {}

resource "shireesh_simple" "example" {
  score = 1
}
```

## Development

- Build/install locally:

```shell
go install
```

- Generate code/docs/examples:

```shell
make generate
```

- Run tests:

```shell
make test
```
