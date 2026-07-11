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

resource "computesphere_guardrail" "example" {
  name        = "example-guardrail"
  description = "A sample guardrail for testing"
  effect      = "block"
  message     = "This is a test guardrail."
  scope       = "project"
  status      = true
  type        = "custom"
}

output "guardrail_id" {
  value = computesphere_guardrail.example.id
}

output "guardrail_name" {
  value = computesphere_guardrail.example.name
} 