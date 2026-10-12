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

resource "computesphere_worker" "jobs" {
  service_id     = var.service_id
  environment_id = var.environment_id
  project_id     = var.project_id

  image        = "acme/app:2.4.0"
  command      = "bin/worker --queue default"
  sphere_count = 2

  env_vars = {
    QUEUE_CONCURRENCY = "8"
  }
}

variable "service_id" {
  description = "Service this resource runs"
  type        = string
}

variable "environment_id" {
  description = "Environment to run it in"
  type        = string
}

variable "project_id" {
  description = "Project the service belongs to"
  type        = string
}

# Moving from computesphere_deployment: replace the old resource block with
# this one and keep a moved block. Terraform carries the deployment over
# without redeploying it.
moved {
  from = computesphere_deployment.jobs
  to   = computesphere_worker.jobs
}
