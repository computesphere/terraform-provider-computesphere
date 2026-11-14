variable "stripe_key" {
  type      = string
  sensitive = true
}

# Shared secrets every deployment in the environment inherits. Values are
# write-only: the API never returns them. This resource owns the set of names.
resource "computesphere_environment_secrets" "example" {
  environment_id = "e1f2a3b4-5678-90ab-cdef-1234567890ab"

  secrets = {
    STRIPE_KEY = var.stripe_key
  }
}
