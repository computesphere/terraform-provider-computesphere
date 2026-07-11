resource "computesphere_project" "example" {
  name        = "tf-datasource-test"
  description = "Project for datasource test"
}

data "computesphere_project" "example" {
  id = computesphere_project.example.id
}

output "project_ds_name" {
  value = data.computesphere_project.example.name
}

output "project_ds_description" {
  value = data.computesphere_project.example.description
}
