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

resource "computesphere_notification_setting" "example" {
  email_enabled = true
  emails        = ["user@example.com"]
}

output "notification_setting_id" {
  value = computesphere_notification_setting.example.id
}

output "notification_setting_email_enabled" {
  value = computesphere_notification_setting.example.email_enabled
} 