resource "computesphere_project" "example" {
  name        = "tf-test-database"
  description = "Database acceptance test"
}

resource "computesphere_environment" "example" {
  name       = "tf-test-db-env"
  region     = "us-east-1"
  project_id = computesphere_project.example.id
}

resource "computesphere_database" "example" {
  name           = "tf-test-db"
  environment_id = computesphere_environment.example.id
  instance_type  = "pg-1c-2g"
  storage_gb     = 10
  inject         = false
}
