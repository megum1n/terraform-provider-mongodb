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

# index
resource mongodb_index "users_index" {
  database = var.database_name
  collection = "collection_name"
  name = "index_name"
  keys = [{
    field = "id"
    type = "1"
  }]

}