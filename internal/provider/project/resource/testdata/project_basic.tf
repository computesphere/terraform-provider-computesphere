resource "computesphere_project" "example" {
  name        = "tf-test-project"
  description = "Test project from acceptance test"
  plan_name   = "MAX"
  plan_value  = 2
}