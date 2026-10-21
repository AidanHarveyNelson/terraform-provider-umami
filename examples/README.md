# Examples

This directory contains examples that are mostly used for documentation, but can also be run/tested manually via the Terraform CLI.

The document generation tool looks for files in the following locations by default. All other \*.tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating examples that can run and/or are testable even if some parts are not relevant for the documentation.

- **provider/provider.tf** example file for the provider index page
- **data-sources/`full data source name`/data-source.tf** example file for the named data source page
- **resources/`full resource name`/resource.tf** example file for the named resource page

## Local smoke testing

Terraform must be pointed at a locally built provider binary through a development override, because `aidanharveynelson/umami` cannot be resolved from the registry until it is published. The repository provides a target for this:

```sh
make umami-up   # optional: start a throwaway Umami and write .env for it
make example-smoke EXAMPLE=examples/resources/umami_website
```

The target builds the provider into `.tmp/`, then runs `apply`, a `plan` that must be empty, and `destroy` against the instance configured in `.env`.

## Example authoring rules

Each example directory must remain a valid Terraform module as a whole.

- Define `terraform.required_providers` only once per directory.
- Define the default `provider "umami"` block only once per directory.
