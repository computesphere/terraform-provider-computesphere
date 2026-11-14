resource "computesphere_project" "example" {
  name        = "tf-test-queue"
  description = "Queue acceptance test"
}

resource "computesphere_environment" "example" {
  name       = "tf-test-queue-env"
  region     = "us-east-1"
  project_id = computesphere_project.example.id
}

resource "computesphere_queue" "example" {
  name           = "tf-test-queue"
  environment_id = computesphere_environment.example.id
  instance_type  = "sq-1c-2g-s20g"
}
