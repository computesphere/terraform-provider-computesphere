data "computesphere_cache_connection" "example" {
  cache_id = "c1d2e3f4-5678-90ab-cdef-1234567890ab"
}

# Host and port, for an app in the same environment.
output "cache_endpoint" {
  value = "${data.computesphere_cache_connection.example.host}:${data.computesphere_cache_connection.example.port}"
}
