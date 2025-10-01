
variable "tf_state_project_id" {
  type        = string
}

variable "region" {
  type        = string
}

variable "hub_project_id" {
  description = "Project ID for the hub connectivity project"
  type        = string
}

# variable "folder_id" {
#   description = "Folder ID where projects will be created"
#   type        = string
# }

variable "billing_account" {
  description = "Billing account ID"
  type        = string
}

variable "cost_center" {
  description = "Cost center for billing allocation"
  type        = string
  default     = "IT-INFRASTRUCTURE"
}

variable "default_region" {
  description = "Default region for resources"
  type        = string
  default     = "asia-southeast2"
  validation {
    condition = contains([
      "asia-southeast1", "asia-southeast2", "asia-east1", "asia-south1",
      "us-central1", "us-east1", "us-west1", "europe-west1"
    ], var.default_region)
    error_message = "Region must be a valid GCP region."
  }
}

variable "hub_cidr_range" {
  description = "CIDR range for the hub VPC"
  type        = string
  default     = "10.0.0.0/16"
  validation {
    condition     = can(cidrhost(var.hub_cidr_range, 0))
    error_message = "Hub CIDR range must be a valid CIDR block."
  }
}

variable "hub_subnets" {
  description = "Hub VPC subnets configuration"
  type = map(object({
    ip_range = string
    region   = string
  }))
  default = {
    "hub-management" = {
      ip_range = "10.0.1.0/24"
      region   = "asia-southeast2"
    }
    "hub-shared-services" = {
      ip_range = "10.0.2.0/24"
      region   = "asia-southeast2"
    }
    "hub-firewall" = {
      ip_range = "10.0.10.0/24"
      region   = "asia-southeast2"
    }
  }
}

variable "nat_ip_count" {
  description = "Number of NAT IP addresses for hub VPC"
  type        = number
  default     = 1
}

variable "enable_private_dns" {
  description = "Enable private DNS zone"
  type        = bool
  default     = true
}

variable "domain_name" {
  description = "Domain name for private DNS zone"
  type        = string
  default     = "company.local"
}

variable "enable_private_google_access" {
  description = "Enable Private Google Access for subnets"
  type        = bool
  default     = true
}

variable "enable_cloud_armor" {
  description = "Enable Cloud Armor security policies"
  type        = bool
  default     = true
}
