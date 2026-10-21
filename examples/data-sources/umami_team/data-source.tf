data "umami_team" "client" {
  name = "Example Client"
}

output "client_viewers" {
  value = [for m in data.umami_team.client.members : m.username if m.role == "team-view-only"]
}
