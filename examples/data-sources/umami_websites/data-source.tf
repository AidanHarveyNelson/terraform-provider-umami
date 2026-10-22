data "umami_websites" "client" {
  team_id = "5e1ba5a1-263f-4edc-ba3a-988342134d17"
}

data "umami_websites" "staging" {
  search = "staging."
}

output "client_website_ids" {
  value = { for w in data.umami_websites.client.websites : w.domain => w.id }
}
