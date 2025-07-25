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

resource "computesphere_environment" "example" {
  name       = "example-environment"
  region     = "us-east-1"
  project_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890ab"
}

output "environment_id" {
  value = computesphere_environment.example.id
}

output "environment_name" {
  value = computesphere_environment.example.name
}

output "environment_region" {
  value = computesphere_environment.example.region
}

output "environment_project_id" {
  value = computesphere_environment.example.project_id
} 