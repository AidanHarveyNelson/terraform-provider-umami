# Umami Terraform Provider

The [Umami](https://umami.is/) provider for Terraform. It manages websites, teams, team members, share links, tracked links, pixels and users through the Umami REST API, on self-hosted instances and on Umami Cloud.

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

The environment variable names match the official API client.

When both an API key and a username/password are set, the provider tries the API key first and logs in if Umami rejects it. Instances older than v3.4.0 always reject API keys. So one configuration keeps working before and after an upgrade to v3.4.0.

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
| Resource | [`umami_website_share`](docs/resources/website_share.md) |
| Resource | [`umami_team`](docs/resources/team.md) |
| Resource | [`umami_team_member`](docs/resources/team_member.md) |
| Resource | [`umami_link`](docs/resources/link.md) |
| Resource | [`umami_pixel`](docs/resources/pixel.md) |
| Resource | [`umami_user`](docs/resources/user.md) (admin only) |
| Data source | [`umami_website`](docs/data-sources/website.md) |
| Data source | [`umami_websites`](docs/data-sources/websites.md) |
| Data source | [`umami_team`](docs/data-sources/team.md) |
| Data source | [`umami_me`](docs/data-sources/me.md) |
| Data source | [`umami_user`](docs/data-sources/user.md) (admin only) |
| Data source | [`umami_config`](docs/data-sources/config.md) |

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

To run them against a throwaway local instance instead, start Umami with Docker and let it write `.env` for a non-admin user, plus the admin login the `umami_user` tests switch to:

```sh
make umami-up                   # Umami 3.4.0, API key auth
make testacc
make umami-down
```

`make umami-up` takes three options:

- `AUTH` sets the authentication mode. `api_key` (default) uses a real API key and needs v3.4.0+. `login` uses a username and password. `fallback` sets an invalid API key plus a username and password, to test the fallback to login.
- `UMAMI_VERSION` sets the image tag, e.g. `UMAMI_VERSION=3.3.1 AUTH=login` for an instance from before API keys existed.
- `UMAMI_PORT` sets the local port (default `3000`).

CI uses the same docker compose setup, so it needs no secrets. Each job starts its own Umami, and the matrix covers Umami 3.4.0 and 3.3.1, every auth mode, and Terraform 1.0 and latest.

### Releasing

1. Move the `Unreleased` entries in `CHANGELOG.md` under the new version and merge.
2. Tag `main` and push the tag, e.g. `git tag v0.2.0 && git push origin v0.2.0`.

The release workflow builds the binaries with GoReleaser, signs the checksums and publishes a GitHub release with the registry manifest. It needs the `GPG_PRIVATE_KEY` and `PASSPHRASE` repository secrets. The Terraform Registry picks up new releases through its GitHub webhook once the provider is published there with the matching GPG public key.

CI runs `goreleaser check` on every change, so a broken release config fails before a tag is pushed.
