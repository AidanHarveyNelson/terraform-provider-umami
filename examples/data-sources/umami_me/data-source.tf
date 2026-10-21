data "umami_me" "current" {}

output "automation_user_id" {
  value = data.umami_me.current.id
}
