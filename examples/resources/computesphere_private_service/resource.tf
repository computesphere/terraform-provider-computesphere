terraform {
  required_providers {
    computesphere = {
      source  = "computesphere/computesphere"
      version = "~> 1.5"
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

# A private service is created with its own service: set name, not service_id.
resource "computesphere_private_service" "api" {
  name           = "api"
  environment_id = var.environment_id
  project_id     = var.project_id

  image = "acme/api:2.4.0"
  port  = 9000

  health_check = {
    enabled = true
    path    = "/ready"
  }
}

variable "environment_id" {
  description = "Environment to run it in"
  type        = string
}

variable "project_id" {
  description = "Project the service belongs to"
  type        = string
}

output "internal_url" {
  value = computesphere_private_service.api.internal_url
}

# Moving from computesphere_deployment (Terraform 1.8 or later): replace the
# old resource block with this one and keep a moved block. The move itself
# doesn't redeploy. Settings the old resource didn't track (for example a
# health check, or a cron job's schedule) apply as one in-place update, which
# rolls the service out. Keep service_id in the new block: name would create a
# new service.
moved {
  from = computesphere_deployment.api
  to   = computesphere_private_service.api
}
