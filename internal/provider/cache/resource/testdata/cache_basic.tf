resource "computesphere_project" "example" {
  name        = "tf-test-cache"
  description = "Cache acceptance test"
}

resource "computesphere_environment" "example" {
  name       = "tf-test-cache-env"
  region     = "us-east-1"
  project_id = computesphere_project.example.id
}

resource "computesphere_cache" "example" {
  name           = "tf-test-cache"
  environment_id = computesphere_environment.example.id
  instance_type  = "sc-0.25c-1g"
}
