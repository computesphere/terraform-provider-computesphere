# Shared variables every deployment in the environment inherits. A
# deployment's own value for a key wins. This resource owns the whole set.
resource "computesphere_environment_variables" "example" {
  environment_id = "e1f2a3b4-5678-90ab-cdef-1234567890ab"

  variables = {
    LOG_LEVEL = "info"
    REGION    = "us-east-1"
  }
}
