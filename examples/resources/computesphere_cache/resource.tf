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

data "computesphere_cache_instance_types" "all" {
  environment_id = var.environment_id
}

resource "computesphere_cache" "example" {
  name           = "app-cache"
  environment_id = var.environment_id
  instance_type  = "sc-0.25c-1g" # Cache 1 GB
  eviction       = "allkeys-lru"
}

variable "environment_id" {
  description = "Environment to create the cache in"
  type        = string
}

output "cache_id" {
  value = computesphere_cache.example.id
}

output "cache_status" {
  value = computesphere_cache.example.status
}
