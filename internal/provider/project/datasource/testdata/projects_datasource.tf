resource "computesphere_project" "example1" {
  name        = "tf-datasource-list-1"
  description = "Project 1 for datasource list test"
}

resource "computesphere_project" "example2" {
  name        = "tf-datasource-list-2"
  description = "Project 2 for datasource list test"
}

data "computesphere_projects" "all" {
  depends_on = [computesphere_project.example1, computesphere_project.example2]
}

output "all_project_names" {
  value = [for p in data.computesphere_projects.all.projects : p.name]
}
