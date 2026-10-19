# Umami Terraform Provider

The [Umami](https://umami.is/) provider for Terraform. It manages websites and teams through the Umami REST API, on self-hosted instances and on Umami Cloud.

> **Note:** The provider is built on the [Umami API](https://docs.umami.is/docs/api), using the same endpoints and authentication as the official [`@umami/api-client`](https://docs.umami.is/docs/api/api-client).

## Development

Requires Terraform 1.0+ and Go 1.25+

## Getting started

### Prerequisites

Create a dedicated, non-admin Umami user for automation. Admin credentials can read and delete every website on the instance.

| Umami version | Authentication |
| --- | --- |
| Self-hosted v3.4.0+ | API key (`umami_...`), created under Settings → API keys |
| Umami Cloud | API key, with `endpoint = "https://api.umami.is/v1"` |
| Self-hosted < v3.4.0 | Username and password. The user must not have two-factor authentication enabled |

### Installation

```terraform
terraform {
  required_providers {
    umami = {
      source  = "aidanharveynelson/umami"
      version = "~> 0.1"
    }
  }
}

provider "umami" {
  endpoint = "https://stats.example.com/api"
}
```

| Attribute | Environment variable |
| --- | --- |
| `endpoint` | `UMAMI_API_CLIENT_ENDPOINT` |
| `api_key` | `UMAMI_API_KEY` |
| `username` | `UMAMI_USERNAME` |
| `password` | `UMAMI_PASSWORD` |

The environment variable names match the official API client. When both an API key and a username/password are set, the API key is used.

### Usage

```terraform
resource "umami_team" "client" {
  name = "Example Client"
}

resource "umami_website" "app" {
  id      = uuidv5("dns", "app.example.com")
  name    = "Example app"
  domain  = "app.example.com"
  team_id = umami_team.client.id
}
```

`id` is optional. Setting it makes the website ID known at plan time, so it can be passed to the frontend build (for `data-website-id`) in the same apply. Use `uuidv5()`, not `random_uuid`: Umami accepts only RFC 4122 UUIDs, and `random_uuid` output often isn't one.

Changing `team_id` transfers the website and keeps its data. Changing `id` replaces the website, which loses its analytics history.

### Importing existing websites

Import existing websites by ID so their history is kept:

```sh
terraform import umami_website.app a8729a96-0000-4000-8000-000000000000
```

## Resources and data sources

| Type | Name |
| --- | --- |
| Resource | [`umami_website`](docs/resources/website.md) |
| Resource | [`umami_team`](docs/resources/team.md) |
| Data source | [`umami_website`](docs/data-sources/website.md) |

## Developing the provider

```sh
make build      # go build
make test       # unit tests
make lint       # golangci-lint
make docs       # regenerate docs/ from schemas, templates and examples
```

### Acceptance tests

Acceptance tests create and destroy real objects in the Umami instance configured in `.env` (see `.env.example`):

```sh
make testacc
```

To run them against a throwaway local instance instead, start Umami with Docker and let it write `.env` for a non-admin user with an API key:

```sh
make umami-up
make testacc
make umami-down
```

CI runs the acceptance tests when the `UMAMI_API_CLIENT_ENDPOINT` and `UMAMI_API_KEY` repository secrets are set, and skips them otherwise.

### Releasing

Push a `v*` tag. The release workflow builds and signs the binaries with GoReleaser. It needs the `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets.
