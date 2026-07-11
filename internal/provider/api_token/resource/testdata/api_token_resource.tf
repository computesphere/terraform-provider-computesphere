resource "computesphere_api_token" "example" {
  name   = "example-token"
  scope  = "full"
  expiry = "2027-01-01T00:00:00Z"
}
