variable "mongo_host" {
  description = "MongoDB hostname"
  type        = string
  default     = "<host>:27017"
}

variable "mongo_username" {
  description = "MongoDB admin username"
  type        = string
}

variable "mongo_password" {
  description = "MongoDB admin password"
  type        = string
}

variable "database_name" {
  description = "Database name"
  type        = string
}

variable "role_name" {
  description = "MongoDB role name"
  type        = string
}

variable "user_username" {
  description = "New MongoDB username"
  type        = string
}

variable "user_password" {
  description = "New MongoDB user password"
  type        = string
}
