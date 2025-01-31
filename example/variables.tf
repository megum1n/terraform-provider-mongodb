variable "mongo_hosts" {
  description = "MongoDB hostname"
  type        = list(string)
  default     = ["localhost:27017"]
}

variable "tls" {
  description = "Enable TLS"
  type        = bool
  default     = false
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
