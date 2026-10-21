data "umami_website" "by_id" {
  id = "0f3c9a3e-7f4d-5b8a-9c1e-2d3f4a5b6c7d"
}

# Find the website ID for a frontend build without hard-coding it.
data "umami_website" "app" {
  domain = "app.example.com"
}

output "website_id" {
  value = data.umami_website.app.id
}
