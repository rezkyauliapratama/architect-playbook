module "spoke_nonprod_project" {
  source = "../../../modules/project"
  
  project_name    = "Spoke Non-Production"
  project_id      = var.spoke_nonprod_project_id
  billing_account = var.billing_account
  environment     = "nonprod"
  enable_org_policies = false

  required_apis = [
    "compute.googleapis.com",
    "container.googleapis.com",
    "servicenetworking.googleapis.com",
    "networkconnectivity.googleapis.com"  # NCC API
  ]
  
  labels = {
    project-type = "connectivity"
    tier        = "spoke"
    environment = "nonprod"
  }
}

module "spoke_nonprod_network" {
  source = "../../../modules/network"
  
  project_id     = module.spoke_nonprod_project.project_id
  network_name   = "spoke-nonprod-vpc"
  environment    = "nonprod"
  default_region = var.default_region
  enable_nat     = false  # NAT provided by hub
  subnets        = var.spoke_nonprod_subnets
  
  depends_on = [module.spoke_nonprod_project]
}
