variable "project_name" {
  description = "Display name for the project"
  type        = string
}

variable "project_id" {
  description = "Unique project ID"
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Project ID must be 6-30 characters, start with a letter, and contain only lowercase letters, numbers, and hyphens."
  }
}

# variable "folder_id" {
#   description = "Folder ID to create project in"
#   type        = string
#   validation {
#     condition     = can(regex("^[0-9]+$", var.folder_id))
#     error_message = "Folder ID must be numeric."
#   }
# }

variable "billing_account" {
  description = "Billing account ID"
  type        = string
  validation {
    condition     = can(regex("^[A-Z0-9]{6}-[A-Z0-9]{6}-[A-Z0-9]{6}$", var.billing_account))
    error_message = "Billing account must be in format XXXXXX-XXXXXX-XXXXXX."
  }
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "nonprod"
  validation {
    condition = contains(["shared", "nonprod", "prod", "uat", "sit", "dev"], var.environment)
    error_message = "Environment must be one of: shared, nonprod, prod, uat, sit, dev."
  }
}

variable "required_apis" {
  description = "List of APIs to enable"
  type        = list(string)
  default = [
    "compute.googleapis.com",
    "servicenetworking.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com"
  ]
}

variable "labels" {
  description = "Labels to apply to resources"
  type        = map(string)
  default     = {}
}

variable "enable_org_policies" {
  description = "Enable organization policies for banking compliance"
  type        = bool
  default     = true
}
