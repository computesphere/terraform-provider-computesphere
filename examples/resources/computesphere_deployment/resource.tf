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

variable "registry_password" {
  description = "Private registry password / access token"
  type        = string
  sensitive   = true
}

# Public image, optionally with public docker build args.
resource "computesphere_deployment" "example" {
  service_id     = "s1a2b3c4-5678-90ab-cdef-1234567890ab"
  environment_id = "e1f2a3b4-5678-90ab-cdef-1234567890ab"
  project_id     = "a1b2c3d4-e5f6-7890-abcd-ef1234567890ab"
  type           = "web-service"

  image        = "nginx:latest"
  port         = 80
  sphere_count = 2

  # PUBLIC docker build args — baked into image layers, so never secrets.
  build_args = {
    NEXT_PUBLIC_API_URL = "https://api.example.com"
  }

  env_vars = {
    LOG_LEVEL = "info"
  }

  secret_vars = {
    API_KEY = "example-secret-value"
  }
}

# Private image pulled from an authenticated registry.
resource "computesphere_deployment" "private" {
  service_id     = "s2b3c4d5-6789-01bc-def0-234567890abc"
  environment_id = "e1f2a3b4-5678-90ab-cdef-1234567890ab"
  project_id     = "a1b2c3d4-e5f6-7890-abcd-ef1234567890ab"
  type           = "web-service"

  image          = "myregistry.azurecr.io/app:1.2.3"
  image_type     = "private"
  image_provider = "azure"
  image_url      = "myregistry.azurecr.io"
  image_username = "service-principal-id"
  image_password = var.registry_password
  port           = 8080
}

output "deployment_id" {
  value = computesphere_deployment.example.id
}

output "deployment_status" {
  value = computesphere_deployment.example.status
}
