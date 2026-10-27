# Changelog

All notable changes to this provider are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `umami_config` data source, for the server settings such as the tracking
  script name.
- `umami_user` resource and data source, for admins. They need username and
  password authentication, because Umami refuses API keys for users.

## [0.1.0] - 2026-10-22

### Added

- Provider configuration with API key or username/password authentication.
  When both are set, the provider falls back to login if the API key is rejected
  (Umami before v3.4.0).
- `umami_website` resource, with a caller-chosen `id`, team transfer and
  session replay and heatmap settings (`replay_config`).
- `umami_team` resource.
- `umami_team_member` resource, to add users to a team with a role.
- `umami_website_share` resource, for public dashboard links.
- `umami_website` data source, looked up by `id` or `domain`.
- `umami_websites`, `umami_team` and `umami_me` data sources.

[Unreleased]: https://github.com/AidanHarveyNelson/terraform-provider-umami/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/AidanHarveyNelson/terraform-provider-umami/releases/tag/v0.1.0
