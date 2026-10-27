# Managing users needs an admin login; Umami refuses API keys for users.
provider "umami" {
  alias    = "admin"
  username = "admin"
  password = var.umami_admin_password
}

variable "umami_admin_password" {
  type      = string
  sensitive = true
}

variable "client_password" {
  type      = string
  sensitive = true
}

resource "umami_user" "client" {
  provider = umami.admin

  username = "client-example"
  password = var.client_password
  role     = "view-only"
}

resource "umami_team" "client" {
  name = "Example client"
}

resource "umami_team_member" "client" {
  team_id = umami_team.client.id
  user_id = umami_user.client.id
  role    = "team-view-only"
}
