resource "computesphere_alert" "example" {
  project_id        = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  environment_id    = "b3669d6a-d89e-4037-873a-65903f3f3ced"
  alert_type        = "cpu"
  severity          = "high"
  threshold         = 80
  evaluation_period = 5
  active            = true
}

data "computesphere_alert" "example" {
  id = computesphere_alert.example.id
}
