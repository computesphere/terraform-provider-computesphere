data "computesphere_queue_connection" "example" {
  id = "c1d2e3f4-5678-90ab-cdef-1234567890ab"
}

# Host and port, for an app in the same environment.
output "queue_endpoint" {
  value = "${data.computesphere_queue_connection.example.host}:${data.computesphere_queue_connection.example.port}"
}
