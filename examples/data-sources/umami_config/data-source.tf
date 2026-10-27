data "umami_config" "server" {}

resource "umami_website" "app" {
  name   = "Example app"
  domain = "app.example.com"
}

output "tracking_code" {
  value = "<script defer src=\"https://analytics.example.com/${data.umami_config.server.tracker_script_name}\" data-website-id=\"${umami_website.app.id}\"></script>"
}
