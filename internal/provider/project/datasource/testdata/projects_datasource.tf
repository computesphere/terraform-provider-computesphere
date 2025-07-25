resource "computesphere_project" "example1" {
  name        = "tf-datasource-list-1"
  description = "Project 1 for datasource list test"
  plan_name   = "MAX"
  plan_value  = 2
}

resource "computesphere_project" "example2" {
  name        = "tf-datasource-list-2"
  description = "Project 2 for datasource list test"
  plan_name   = "MAX"
  plan_value  = 2
}

data "computesphere_projects" "all" {}

output "all_project_names" {
  value = [for p in data.computesphere_projects.all.projects : p.name]
} 