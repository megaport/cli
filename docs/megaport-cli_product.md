# product

Manage products in the Megaport API

## Description

Manage products in the Megaport API.

This command groups operations related to products. You can use the subcommands to list all products, get the type of a specific product, or list the products your company can order.

### Example Usage

```sh
  megaport-cli product list
  megaport-cli product get-type [productUID]
  megaport-cli product availability
```

## Usage

```sh
megaport-cli product [flags]
```


## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|

## Subcommands
* [availability](megaport-cli_product_availability.md)
* [get-type](megaport-cli_product_get-type.md)
* [list](megaport-cli_product_list.md)

