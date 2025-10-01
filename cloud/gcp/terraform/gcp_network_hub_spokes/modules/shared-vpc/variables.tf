variable "host_project_id" {
  description = "Project ID of the shared VPC host project"
  type        = string
}

variable "service_project_ids" {
  description = "List of service project IDs to attach to the shared VPC"
  type        = list(string)
  default     = []
}

variable "shared_vpc_admins" {
  description = "List of members who will have shared VPC admin role"
  type        = list(string)
  default     = []
}

variable "subnet_users" {
  description = "Map of subnet users with their permissions"
  type = map(object({
    region     = string
    subnetwork = string
    members    = list(string)
  }))
  default = {}
}
