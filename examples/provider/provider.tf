provider "umami" {
  endpoint = "https://stats.example.com/api" # Also configurable via UMAMI_API_CLIENT_ENDPOINT
  api_key  = var.umami_api_key               # Also configurable via UMAMI_API_KEY

  # On instances older than v3.4.0, log in instead (the user must not have 2FA):
  # username = "automation" # Also configurable via UMAMI_USERNAME
  # password = "..."        # Also configurable via UMAMI_PASSWORD
}

variable "umami_api_key" {
  type      = string
  sensitive = true
}
