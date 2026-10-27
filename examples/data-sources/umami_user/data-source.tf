# Looking up users needs an admin login; Umami refuses API keys for users.
provider "umami" {
  alias    = "admin"
  username = "admin"
  password = var.umami_admin_password
}

variable "umami_admin_password" {
  type      = string
  sensitive = true
}

data "umami_user" "jane" {
  provider = umami.admin
  username = "jane"
}

resource "umami_team" "client" {
  name = "Example client"
}

resource "umami_team_member" "jane" {
  team_id = umami_team.client.id
  user_id = data.umami_user.jane.id
  role    = "team-member"
}
