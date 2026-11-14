data "computesphere_database_connection" "example" {
  database_id = "d1e2f3a4-5678-90ab-cdef-1234567890ab"
}

# Host and port of the pooled route, for an app in the same environment.
output "pooled_endpoint" {
  value = "${data.computesphere_database_connection.example.pooled.host}:${data.computesphere_database_connection.example.pooled.port}"
}
