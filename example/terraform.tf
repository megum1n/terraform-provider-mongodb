
provider "mongodb" {
  hosts    = [var.mongo_host]
  username = var.mongo_username
  password = var.mongo_password
  # tls = true # optional
}

terraform {
  required_providers {
    mongodb = {
      source  = "megum1n/mongodb"
      version = "0.2.7"
    }
  }
}