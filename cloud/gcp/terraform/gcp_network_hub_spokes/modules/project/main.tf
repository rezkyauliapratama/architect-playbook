resource "google_project" "project" {
  name                = var.project_name
  project_id          = var.project_id
  # folder_id           = var.folder_id
  billing_account     = var.billing_account
  auto_create_network = false

  labels = merge(var.labels, {
    environment = var.environment
  })
}

resource "google_project_service" "required_apis" {
  for_each = toset(var.required_apis)
  
  project = google_project.project.project_id
  service = each.value
  
  disable_dependent_services = true
  disable_on_destroy        = false
}

# Organization policies for banking compliance
resource "google_project_organization_policy" "vm_external_ip_access" {
  count      = var.enable_org_policies ? 1 : 0
  project    = google_project.project.project_id
  constraint = "compute.vmExternalIpAccess"

  list_policy {
    deny {
      all = true
    }
  }
}

resource "google_project_organization_policy" "require_ssl_certificates" {
  count      = var.enable_org_policies ? 1 : 0
  project    = google_project.project.project_id
  constraint = "compute.requireSslCertificates"

  boolean_policy {
    enforced = false #set value to true if u want to force SSL communication
  }
}

resource "google_project_organization_policy" "skip_default_network_creation" {
  count      = var.enable_org_policies ? 1 : 0
  project    = google_project.project.project_id
  constraint = "compute.skipDefaultNetworkCreation"

  boolean_policy {
    enforced = true
  }
}
