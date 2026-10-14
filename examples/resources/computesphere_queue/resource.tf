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

data "computesphere_queue_instance_types" "all" {
  environment_id = var.environment_id
}

resource "computesphere_queue" "example" {
  name           = "app-queue"
  environment_id = var.environment_id
  instance_type  = "sq-1c-2g-s20g" # Queue 1: 1 vCPU, 2 GB, 20 GB disk
}

variable "environment_id" {
  description = "Environment to create the broker in"
  type        = string
}

output "queue_id" {
  value = computesphere_queue.example.id
}

output "queue_instances" {
  value = computesphere_queue.example.instances
}
