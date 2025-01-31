provider "mongodb" {
  hosts    = var.mongo_hosts
  username = var.mongo_username
  password = var.mongo_password
  tls      = var.tls
}

terraform {
  required_providers {
    mongodb = {
      source = "megum1n/mongodb"
    }
  }
}
