# Enable shared VPC host
resource "google_compute_shared_vpc_host_project" "host" {
  project = var.host_project_id
}

# Attach service projects
resource "google_compute_shared_vpc_service_project" "service" {
  for_each = toset(var.service_project_ids)
  
  host_project    = google_compute_shared_vpc_host_project.host.project
  service_project = each.value
}

# IAM binding for service project admins
resource "google_project_iam_binding" "shared_vpc_admin" {
  count   = length(var.shared_vpc_admins) > 0 ? 1 : 0
  project = var.host_project_id
  role    = "roles/compute.xpnAdmin"
  members = var.shared_vpc_admins
}

# Subnet-level IAM for service projects
resource "google_compute_subnetwork_iam_binding" "service_project_users" {
  for_each = var.subnet_users
  
  project    = var.host_project_id
  region     = each.value.region
  subnetwork = each.value.subnetwork
  role       = "roles/compute.networkUser"
  members    = each.value.members
}

# Security admin role for shared VPC
resource "google_project_iam_binding" "shared_vpc_security_admin" {
  count   = length(var.shared_vpc_admins) > 0 ? 1 : 0
  project = var.host_project_id
  role    = "roles/compute.securityAdmin"
  members = var.shared_vpc_admins
}
