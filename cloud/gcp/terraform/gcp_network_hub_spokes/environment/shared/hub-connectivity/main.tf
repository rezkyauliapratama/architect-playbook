module "hub_project" {
  source = "../../../modules/project"
  
  project_name    = "Hub Connectivity"
  project_id      = var.hub_project_id
  # folder_id       = var.folder_id
  billing_account = var.billing_account
  environment     = "shared"
  enable_org_policies = false
  
  required_apis = [
    "compute.googleapis.com",
    "servicenetworking.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "dns.googleapis.com"
  ]
  
  labels = {
    project-type = "connectivity"
    tier        = "hub"
    compliance  = "banking"
  }
}

module "hub_network" {
  source = "../../../modules/network"
  
  project_id     = module.hub_project.project_id
  network_name   = "hub-vpc"
  environment    = "shared"
  default_region = var.default_region
  enable_nat     = true
  nat_ip_count   = var.nat_ip_count
  subnets        = var.hub_subnets
  
  depends_on = [module.hub_project]
}

# Private DNS zone for internal resolution
resource "google_dns_managed_zone" "private_zone" {
  count       = var.enable_private_dns ? 1 : 0
  project     = module.hub_project.project_id
  name        = "private-zone"
  dns_name    = "internal.${var.domain_name}."
  description = "Private DNS zone for internal services"
  
  visibility = "private"
  
  private_visibility_config {
    networks {
      network_url = module.hub_network.network_self_link
    }
  }
}
