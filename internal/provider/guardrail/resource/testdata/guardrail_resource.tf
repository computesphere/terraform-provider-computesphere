resource "computesphere_guardrail" "example" {
  name        = "example-guardrail"
  description = "A sample guardrail for testing"
  effect      = "block"
  message     = "This is a test guardrail."
  scope       = "project"
  status      = true
  type        = "custom"
}
