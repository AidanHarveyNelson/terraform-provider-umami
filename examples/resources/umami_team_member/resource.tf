resource "umami_team" "client" {
  name = "Example Client"
}

# Users are created by an Umami admin; reference them by ID.
variable "client_viewer_user_ids" {
  type    = set(string)
  default = []
}

resource "umami_team_member" "developer" {
  team_id = umami_team.client.id
  user_id = "8c3e2f1a-5b6d-4e7f-9a0b-1c2d3e4f5a6b"
  role    = "team-manager"
}

resource "umami_team_member" "client_viewer" {
  for_each = var.client_viewer_user_ids

  team_id = umami_team.client.id
  user_id = each.value
  role    = "team-view-only"
}
