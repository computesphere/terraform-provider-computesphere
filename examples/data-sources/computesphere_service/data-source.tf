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

data "computesphere_service" "example" {
  id = "s1a2b3c4-5678-90ab-cdef-1234567890ab"
}

output "service" {
  value = data.computesphere_service.example
} 