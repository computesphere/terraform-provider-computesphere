resource "computesphere_service" "example" {
  name       = "example-service"
  project_id = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  type       = "web-service"
  plan_id    = "9dd4b690-f41a-4d6f-b12b-233766628a9b"
}
