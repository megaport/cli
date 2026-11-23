# pricing

Get the monthly price of a product

## Description

Get the monthly price of a new or existing product.

Use the subcommands to price a Port or a VXC.

### Example Usage

```sh
  megaport-cli product pricing port --location-id 1 --speed 1000 --term 12
  megaport-cli product pricing vxc --a-location-id 1 --b-location-id 2 --speed 100
```

## Usage

```sh
megaport-cli product pricing [flags]
```


## Parent Command

* [megaport-cli product](megaport-cli_product.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|

## Subcommands
* [port](megaport-cli_product_pricing_port.md)
* [vxc](megaport-cli_product_pricing_vxc.md)

