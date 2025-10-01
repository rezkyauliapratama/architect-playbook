variable "project_id" {
  description = "The project ID to deploy resources"
  type        = string
}

variable "network_name" {
  description = "Name of the VPC network"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "nonprod"
}

variable "default_region" {
  description = "Default region for resources"
  type        = string
  default     = "asia-southeast2"
}

variable "enable_nat" {
  description = "Enable Cloud NAT for outbound internet access"
  type        = bool
  default     = true
}

variable "nat_ip_count" {
  description = "Number of NAT IP addresses to allocate"
  type        = number
  default     = 1
  validation {
    condition = var.nat_ip_count >= 1 && var.nat_ip_count <= 10
    error_message = "NAT IP count must be between 1 and 10."
  }
}

variable "subnets" {
  description = "Map of subnets to create"
  type = map(object({
    ip_range = string
    region   = string
    secondary_ranges = optional(list(object({
      range_name    = string
      ip_cidr_range = string
    })), [])
  }))
}

variable "enable_flow_logs" {
  description = "Enable VPC flow logs"
  type        = bool
  default     = true
}

variable "flow_log_sampling" {
  description = "Sampling rate for flow logs"
  type        = number
  default     = 0.5
  validation {
    condition = var.flow_log_sampling >= 0.0 && var.flow_log_sampling <= 1.0
    error_message = "Flow log sampling must be between 0.0 and 1.0."
  }
}

variable "routing_mode" {
  description = "Network routing mode"
  type        = string
  default     = "REGIONAL"
  validation {
    condition = contains(["REGIONAL", "GLOBAL"], var.routing_mode)
    error_message = "Routing mode must be REGIONAL or GLOBAL."
  }
}

variable "mtu" {
  description = "Maximum Transmission Unit in bytes"
  type        = number
  default     = 1460
  validation {
    condition = var.mtu >= 1460 && var.mtu <= 8896
    error_message = "MTU must be between 1460 and 8896 bytes."
  }
}
