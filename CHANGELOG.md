# Changelog

All notable changes to this provider are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `umami_team_member` resource, to add users to a team with a role.

## [0.1.0] - 2026-10-22

### Added

- Provider configuration with API key or username/password authentication.
  When both are set, the provider falls back to login if the API key is rejected
  (Umami before v3.4.0).
- `umami_website` resource, with a caller-chosen `id` and team transfer.
- `umami_team` resource.
- `umami_website` data source.

[Unreleased]: https://github.com/AidanHarveyNelson/terraform-provider-umami/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/AidanHarveyNelson/terraform-provider-umami/releases/tag/v0.1.0
