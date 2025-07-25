resource "computesphere_alert" "example" {
  project_id     = "a1b2c3d4-e5f6-7890-abcd-ef1234567890ab"
  environment_id = "e1f2a3b4-5678-90ab-cdef-1234567890ab"
  alert_type     = "cpu"
  severity       = "critical"
  threshold      = 80
  evaluation_period = 5
  active         = true
} 