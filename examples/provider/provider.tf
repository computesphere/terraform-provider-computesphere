provider "computesphere" {
  api_token  = "your-api-token"  # or set COMPUTESPHERE_API_TOKEN
  account_id = "your-account-id" # or set COMPUTESPHERE_ACCOUNT_ID
  # api_url defaults to https://api.computesphere.com/v2 — only set it for
  # non-default topologies, e.g. api_url = "https://api.computesphere.com/v2"
}
