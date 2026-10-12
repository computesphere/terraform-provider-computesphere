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

resource "computesphere_web_service" "app" {
  service_id     = var.service_id
  environment_id = var.environment_id
  project_id     = var.project_id

  image        = "nginx:1.27"
  port         = 8080
  sphere_count = 2

  env_vars = {
    LOG_LEVEL = "info"
  }
  secret_vars = {
    API_KEY = var.api_key
  }

  health_check = {
    enabled = true
    path    = "/healthz"
  }

  autoscaling = {
    enabled         = true
    min_replicas    = 2
    max_replicas    = 6
    cpu_utilization = 70
  }
}

# Built from a git repository.
resource "computesphere_web_service" "web" {
  service_id     = var.web_service_id
  environment_id = var.environment_id
  project_id     = var.project_id

  git_url           = "acme/web"
  branch            = "main"
  git_connection_id = var.git_connection_id
  runtime_type      = "native"
  language          = "nodejs"
  version           = "20"
  build_command     = "npm ci && npm run build"
  start_command     = "npm start"
  port              = 3000
  predeploy_command = "npm run migrate"

  # Public values only: build args are baked into the image.
  build_args = {
    NEXT_PUBLIC_API_URL = "https://api.example.com"
  }
}

variable "api_key" {
  type      = string
  sensitive = true
}

variable "git_connection_id" {
  type = string
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

variable "web_service_id" {
  description = "Service the git-built web service runs"
  type        = string
}

output "url" {
  value = computesphere_web_service.app.url
}

# Moving from computesphere_deployment: replace the old resource block with
# this one and keep a moved block. Terraform carries the deployment over
# without redeploying it.
moved {
  from = computesphere_deployment.app
  to   = computesphere_web_service.app
}
