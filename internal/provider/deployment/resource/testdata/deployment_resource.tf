resource "computesphere_service" "example" {
  name       = "tf-deploy-test-svc"
  project_id = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  type       = "web-service"
  plan_id    = "9dd4b690-f41a-4d6f-b12b-233766628a9b"
}

resource "computesphere_deployment" "example" {
  service_id     = computesphere_service.example.id
  project_id     = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  environment_id = "b3669d6a-d89e-4037-873a-65903f3f3ced"
  type           = "web-service"
  image          = "nginx:latest"
  port           = 80
  wait_for_ready = false
}
