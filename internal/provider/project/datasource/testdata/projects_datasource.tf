resource "computesphere_project" "example1" {
  name        = "tf-datasource-list-1"
  description = "Project for datasource list test"
}

data "computesphere_projects" "all" {
  depends_on = [computesphere_project.example1]
}

output "all_project_names" {
  value = [for p in data.computesphere_projects.all.projects : p.name]
}
