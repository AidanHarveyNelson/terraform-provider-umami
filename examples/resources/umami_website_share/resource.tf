resource "umami_website" "app" {
  name   = "Example app"
  domain = "app.example.com"
}

resource "umami_website_share" "client" {
  website_id   = umami_website.app.id
  name         = "Client dashboard"
  slug         = "example-app"
  views        = ["overview", "events", "realtime"]
  allow_filter = false
}

output "share_url" {
  value = "https://analytics.example.com/share/${umami_website_share.client.slug}"
}
