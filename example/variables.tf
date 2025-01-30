variable "mongo_host" {
  description = "MongoDB hostname"
  type        = string
  default     = "<host>:27017"
}

variable "mongo_username" {
  description = "MongoDB admin username"
  type        = string
  default     = ""
}

variable "mongo_password" {
  description = "MongoDB admin password"
  type        = string
  default     = ""
}

variable "database_name" {
  description = "Database name"
  type        = string
  default     = "test"
}

variable "role_name" {
  description = "MongoDB role name"
  type        = string
  default     = "testRole"
}

variable "user_username" {
  description = "New MongoDB username"
  type        = string
  default     = ""
}

variable "user_password" {
  description = "New MongoDB user password"
  type        = string
  default     = ""
}
