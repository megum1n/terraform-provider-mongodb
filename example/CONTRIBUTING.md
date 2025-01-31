# Contributing

## Test changes locally 

Put this file into your home directory to use local binary

```hcl
provider_installation {
  dev_overrides {
      "megum1n/mongodb" = "~/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Then build and install new binary with

```shell
go install
```

You can run local mongodb instance with docker

```shell
docker compose up -d
```
