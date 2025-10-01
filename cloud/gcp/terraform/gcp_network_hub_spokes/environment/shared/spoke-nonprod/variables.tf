variable "tf_state_project_id" {
  type        = string
}

variable "region" {
  type        = string
}

variable "spoke_nonprod_project_id" {
  description = "Project ID for the spoke non-production project"
  type        = string
}

variable "billing_account" {
  description = "Billing account ID"
  type        = string
}

variable "default_region" {
  description = "Default region for resources"
  type        = string
  default     = "asia-southeast2"
}

variable "spoke_nonprod_cidr_range" {
  description = "CIDR range for the spoke non-prod VPC"
  type        = string
  default     = "10.1.0.0/16"
  validation {
    condition     = can(cidrhost(var.spoke_nonprod_cidr_range, 0))
    error_message = "Spoke non-prod CIDR range must be a valid CIDR block."
  }
}

variable "spoke_nonprod_subnets" {
  description = "Spoke non-prod VPC subnets configuration"
  type = map(object({
    ip_range = string
    region   = string
    secondary_ranges = optional(list(object({
      range_name    = string
      ip_cidr_range = string
    })), [])
  }))
  default = {
    "nonprod-app" = {
      ip_range = "10.1.1.0/24"
      region   = "asia-southeast2"
      secondary_ranges = [
        {
          range_name    = "gke-pods"
          ip_cidr_range = "10.1.16.0/20"
        },
        {
          range_name    = "gke-services"
          ip_cidr_range = "10.1.32.0/20"
        }
      ]
    }
    "nonprod-data" = {
      ip_range = "10.1.2.0/24"
      region   = "asia-southeast2"
    }
  }
}

variable "enable_gke" {
  description = "Enable GKE cluster creation"
  type        = bool
  default     = false
}

variable "enable_binary_authorization" {
  description = "Enable Binary Authorization for container security"
  type        = bool
  default     = true
}
