variable "doppler_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

# A Doppler config that deployments in two projects read secrets from. The
# token is write-only: it is never stored in Terraform state.
resource "computesphere_secret_source" "doppler" {
  name      = "doppler-web"
  type      = "doppler"
  auth_type = "service_token"
  config = {
    project = "web"
    config  = "prd"
  }
  credential         = { token = var.doppler_token }
  credential_version = 1 # bump to send a rotated token
  project_ids = [
    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  ]
}
