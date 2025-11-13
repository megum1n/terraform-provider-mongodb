provider "mongodb" {
  hosts    = var.mongo_hosts
  username = var.mongo_username
  password = var.mongo_password
  tls      = var.tls

  # Optional: Enable direct connection mode (useful for single-node MongoDB deployments)
  # direct_connection = var.direct_connection
}

terraform {
  required_providers {
    mongodb = {
      source = "megum1n/mongodb"
    }
  }
}
