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

data "computesphere_queue_instance_types" "example" {
  environment_id = var.environment_id # optional: check availability for this environment
}

variable "environment_id" {
  description = "Environment to check availability for"
  type        = string
}

output "three_member_queue_sizes" {
  value = [for t in data.computesphere_queue_instance_types.example.instance_types : "${t.name} (${t.id})" if t.available && t.nodes == 3]
}
