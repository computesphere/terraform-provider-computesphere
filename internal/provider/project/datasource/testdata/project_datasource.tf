resource "computesphere_project" "example" {
  name        = "tf-datasource-test"
  description = "Project for datasource test"
  plan_name   = "MAX"
  plan_value  = 2
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

output "project_ds_plan_name" {
  value = data.computesphere_project.example.plan_name
}

output "project_ds_plan_value" {
  value = data.computesphere_project.example.plan_value
} 