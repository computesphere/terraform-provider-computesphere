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

data "computesphere_spherelet_types" "example" {
  region = var.region # optional: check availability in this region
}

variable "region" {
  description = "Region to check availability in"
  type        = string
  default     = "us-east-1"
}

output "available_sizes" {
  value = [for t in data.computesphere_spherelet_types.example.spherelet_types : "${t.name} (${t.id}): ${t.vcpu} vCPU, ${t.memory_mb} MB, $${t.monthly_price_usd}/mo" if t.available]
}
