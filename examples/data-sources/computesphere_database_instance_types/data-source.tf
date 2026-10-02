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

data "computesphere_database_instance_types" "example" {
  environment_id = var.environment_id # optional: price for this environment's region
}

variable "environment_id" {
  description = "Environment whose region to price for"
  type        = string
}

output "available_instance_types" {
  value = [for t in data.computesphere_database_instance_types.example.instance_types : "${t.name} (${t.id}): $${t.monthly_price_cents / 100}/mo" if t.available]
}
