schema_version = 1

project {
  license          = "MPL-2.0"
  copyright_year   = 2026
  copyright_holder = "Aidan Harvey-Nelson"

  header_ignore = [
    # examples used within documentation (prose)
    "examples/**",

    # GitHub issue template configuration
    ".github/ISSUE_TEMPLATE/*.yml",

    # golangci-lint tooling configuration
    ".golangci.yml",

    # GoReleaser tooling configuration
    ".goreleaser.yml",

    # Local acceptance-test tooling
    "docker-compose.yml",
    "scripts/**",
  ]
}
