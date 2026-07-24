terraform {
  required_providers {
    computesphere = {
      source  = "computesphere/computesphere"
      version = "~> 1.1"
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

resource "computesphere_service" "example" {
  name       = "example-service"
  project_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890ab"
  type       = "web-service"
  plan_id    = "f1a2b3c4-5678-90ab-cdef-1234567890ab"
}

output "service_id" {
  value = computesphere_service.example.id
}

output "service_name" {
  value = computesphere_service.example.name
} 