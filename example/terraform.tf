provider "mongodb" {
  hosts    = var.mongo_hosts
  username = var.mongo_username
  password = var.mongo_password
  tls      = var.tls

  # Optional: Enable direct connection mode (useful for single-node MongoDB deployments)
  # direct_connection = var.direct_connection

  # Optional: Specify authentication mechanism and source
  # For AWS IAM authentication:
  # auth_mechanism = "MONGODB-AWS"
  # auth_source    = "$external"
  # username and password can be omitted when using AWS IAM
}

terraform {
  required_providers {
    mongodb = {
      source = "megum1n/mongodb"
    }
  }
}
