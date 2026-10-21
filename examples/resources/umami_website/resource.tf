resource "umami_team" "client" {
  name = "Example Client"
}

# The id is derived from the domain, so it is known at plan time and can be
# passed to the frontend build (data-website-id) in the same apply.
resource "umami_website" "example" {
  id      = uuidv5("dns", "app.example.com")
  name    = "Example app"
  domain  = "app.example.com"
  team_id = umami_team.client.id

  replay_config = {
    replay_enabled  = true
    heatmap_enabled = true
    sample_rate     = 0.1
    mask_level      = "strict"
    block_selector  = "[data-private]"
  }
}

output "website_id" {
  value = umami_website.example.id
}
