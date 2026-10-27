data "umami_config" "server" {}

resource "umami_pixel" "newsletter" {
  name = "Newsletter opens"
  slug = "newsletter-opens"
}

output "pixel_tag" {
  value = "<img src=\"${coalesce(data.umami_config.server.pixels_url, "https://analytics.example.com/p")}/${umami_pixel.newsletter.slug}\" width=\"1\" height=\"1\" alt=\"\">"
}
