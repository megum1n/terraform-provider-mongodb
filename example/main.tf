
provider "mongodb" {

  hosts = ["<host>:27017"] # Change with your hostname
  username = "" #admin default
  password = "" #access user password
  #tls = true # optional

}

terraform {
  required_providers {
    mongodb = {
      source = "megum1n/mongodb"
      version = "0.2.7" # latest currently
     
    }
  }
}


resource "mongodb_role" "example_role" {

  name     = "testRole"
  database = "test"
  privileges = [
    {
      actions = ["find","insert","update","remove"]
      resource = {
        collection = ""
        db = "test"
      }
    }
  ]

  roles      = [
    {
      role = "readWrite"
      db = "test"
    }
  ]
}

resource "mongodb_user" "example_role_user" {

  username = "" #add new user 
  password = "" #add new user password 
  database = "test"

  roles  = [ {
    role = "testRole"
    db   = "test" 
  } ]
  depends_on = [mongodb_role.example_role]
}
