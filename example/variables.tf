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

variable "direct_connection" {
  description = "Direct connection to MongoDB (useful for single-node deployments)"
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

### variables for index

variable "collection_name" {
  description = "Collection name"
  type        = string
}

variable "index_name" {
  description = "Index name"
  type        = string
}

variable "index_keys" {
  description = "Index keys configuration as key-value map (field = index_type)"
  type        = map(string)
}

variable "index_unique" {
  description = "Whether index should be unique"
  type        = bool
  default     = false
}

variable "expire_after_seconds" {
  description = "TTL value in seconds"
  type        = number
  default     = null
}

variable "partial_filter_expression" {
  description = "Partial filter expression"
  type        = map(string)
  default     = null
}

variable "wildcard_projection" {
  description = "Wildcard projection configuration"
  type        = map(number)
  default     = null
}

variable "collation" {
  description = "Collation configuration"
  type = object({
    locale           = string
    strength         = optional(number)
    case_level       = optional(bool)
    case_first       = optional(string)
    numeric_ordering = optional(bool)
    alternate        = optional(string)
    max_variable     = optional(string)
    backwards        = optional(bool)
  })
  default = null
}

variable "weights" {
  description = "Text index weights"
  type        = map(number)
  default     = null
}

variable "default_language" {
  description = "Default language for text index"
  type        = string
  default     = null
}

variable "language_override" {
  description = "Language override field"
  type        = string
  default     = null
}

variable "text_index_version" {
  description = "Text index version"
  type        = number
  default     = null
}

variable "sparse" {
  description = "Whether index should be sparse"
  type        = bool
  default     = false
}

variable "bits" {
  description = "Bits precision for 2d index"
  type        = number
  default     = null
}

variable "min" {
  description = "Minimum value for 2d index"
  type        = number
  default     = null
}

variable "max" {
  description = "Maximum value for 2d index"
  type        = number
  default     = null
}