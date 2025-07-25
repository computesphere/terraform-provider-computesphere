terraform {
  required_providers {
    computesphere = {
      source = "computesphere.com/computesphere/computesphere"
    }
  }
}

variable "api_token" {
  description = "API token for ComputeSphere"
  type        = string
  sensitive   = true
}

variable "account_id" {
  description = "Account ID for ComputeSphere"
  type        = string
}

variable "api_url" {
  description = "API URL for ComputeSphere"
  type        = string
  default     = "api.computesphere.com"
}

provider "computesphere" {
  api_token  = var.api_token  # or set COMPUTESPHERE_API_TOKEN env variable
  account_id = var.account_id # or set COMPUTESPHERE_ACCOUNT_ID env variable
  api_url    = var.api_url    # or set COMPUTESPHERE_API_URL env variable
}

resource "computesphere_project" "example" {
  name        = "tf-example-pre10"
  description = "A sample ComputeSphere project created via Terraform"
  plan_name   = "PWR"
  plan_value  = 50
  # Add other required or optional fields as needed
}

output "project_id" {
  value = computesphere_project.example.id
}

output "project_name" {
  value = computesphere_project.example.name
}

output "project_description" {
  value = computesphere_project.example.description
}

output "project_plan_name" {
  value = computesphere_project.example.plan_name
}

output "project_plan_value" {
  value = computesphere_project.example.plan_value
}

output "project_plan_id" {
  value = computesphere_project.example.plan_id
}

output "project_created_at" {
  value = computesphere_project.example.created_at
}
