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

variable "deployment_id" {
  description = "Deployment to bind the custom domains to"
  type        = string
}

# One resource instance per hostname — for_each keeps each domain individually
# managed and independently statused.
resource "computesphere_custom_domain" "example" {
  for_each = toset([
    "example.com",
    "www.example.com",
  ])

  deployment_id = var.deployment_id
  domain        = each.value
}

output "custom_domain_ids" {
  value = { for k, d in computesphere_custom_domain.example : k => d.id }
}

output "custom_domain_status" {
  value = { for k, d in computesphere_custom_domain.example : k => d.status }
}
