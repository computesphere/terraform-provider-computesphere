resource "computesphere_notification_setting" "example" {
  email_enabled = true
  emails        = ["user@example.com"]
} 