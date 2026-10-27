data "umami_config" "server" {}

resource "umami_team" "marketing" {
  name = "Marketing"
}

resource "umami_link" "launch" {
  name    = "Launch announcement"
  url     = "https://example.com/blog/launch?utm_source=newsletter"
  slug    = "launch-2026"
  team_id = umami_team.marketing.id
}

output "launch_url" {
  value = "${coalesce(data.umami_config.server.links_url, "https://analytics.example.com/q")}/${umami_link.launch.slug}"
}
