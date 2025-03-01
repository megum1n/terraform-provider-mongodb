# role
resource "mongodb_role" "example_role" {
  name     = var.role_name
  database = var.database_name
  privileges = [
    {
      actions = ["find", "insert", "update", "remove"]
      resource = {
        # "" for all collections
        collection = ""
        db         = var.database_name
      }
    }
  ]
  roles = [
    {
      role = "readWrite"
      db   = var.database_name
    }
  ]
}

# user
resource "mongodb_user" "example_role_user" {
  username = var.user_username
  password = var.user_password
  database = var.database_name

  roles = [
    {
      role = var.role_name
      db   = var.database_name
    }
  ]

  depends_on = [mongodb_role.example_role]
}


# index example
# Generic index resource that can be reused
resource "mongodb_index" "example_index" {
  database   = var.database_name
  collection = var.collection_name
  name       = var.index_name
  keys       = var.index_keys
  
  # Optional parameters
  unique                   = var.index_unique
  expire_after_seconds     = var.expire_after_seconds
  partial_filter_expression = var.partial_filter_expression
  wildcard_projection      = var.wildcard_projection
  collation                = var.collation
  weights                  = var.weights
  default_language         = var.default_language
  language_override        = var.language_override
  text_index_version       = var.text_index_version
  sparse                   = var.sparse
  bits                     = var.bits
  min                      = var.min
  max                      = var.max
}