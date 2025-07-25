resource "computesphere_service" "example" {
  name         = "example-service"
  project_id   = "a1b2c3d4-e5f6-7890-abcd-ef1234567890ab"
  env_id       = "e1f2a3b4-5678-90ab-cdef-1234567890ab"
  image_name   = "nginx:latest"
  port         = 80
  sphere_count = 2
  type         = "web"
} 