resource "google_compute_network" "vpc" {
  name                    = var.network_name
  project                 = var.project_id
  auto_create_subnetworks = false
  mtu                     = var.mtu
  routing_mode           = var.routing_mode
  
  description = "VPC for ${var.environment} environment with banking compliance"
}

resource "google_compute_subnetwork" "subnets" {
  for_each = var.subnets
  
  name                     = each.key
  project                  = var.project_id
  network                  = google_compute_network.vpc.id
  ip_cidr_range           = each.value.ip_range
  region                  = each.value.region
  private_ip_google_access = true
  
  dynamic "log_config" {
    for_each = var.enable_flow_logs ? [1] : []
    content {
      aggregation_interval = "INTERVAL_10_MIN"
      flow_sampling       = var.flow_log_sampling
      metadata           = "INCLUDE_ALL_METADATA"
    }
  }

  dynamic "secondary_ip_range" {
    for_each = lookup(each.value, "secondary_ranges", [])
    content {
      range_name    = secondary_ip_range.value.range_name
      ip_cidr_range = secondary_ip_range.value.ip_cidr_range
    }
  }
}

# Cloud Router for NAT
resource "google_compute_router" "router" {
  count   = var.enable_nat ? 1 : 0
  name    = "${var.network_name}-router"
  project = var.project_id
  region  = var.default_region
  network = google_compute_network.vpc.id
}

# Reserved IP addresses for NAT
resource "google_compute_address" "nat_ips" {
  count   = var.enable_nat ? var.nat_ip_count : 0
  name    = "${var.network_name}-nat-ip-${count.index + 1}"
  project = var.project_id
  region  = var.default_region
}

# Cloud NAT for outbound internet access
resource "google_compute_router_nat" "nat" {
  count  = var.enable_nat ? 1 : 0
  name   = "${var.network_name}-nat"
  project = var.project_id
  router = google_compute_router.router[0].name
  region = var.default_region

  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                           = google_compute_address.nat_ips[*].self_link
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

# Banking compliance firewall rules
resource "google_compute_firewall" "deny_all_ingress" {
  name    = "${var.network_name}-deny-all-ingress"
  project = var.project_id
  network = google_compute_network.vpc.name

  direction = "INGRESS"
  priority  = 65534

  deny {
    protocol = "all"
  }

  source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "allow_internal" {
  name    = "${var.network_name}-allow-internal"
  project = var.project_id
  network = google_compute_network.vpc.name

  direction = "INGRESS"
  priority  = 1000

  allow {
    protocol = "tcp"
  }
  allow {
    protocol = "udp"
  }
  allow {
    protocol = "icmp"
  }

  source_ranges = [for subnet in var.subnets : subnet.ip_range]
}

# Allow SSH from IAP
resource "google_compute_firewall" "allow_iap_ssh" {
  name    = "${var.network_name}-allow-iap-ssh"
  project = var.project_id
  network = google_compute_network.vpc.name

  direction = "INGRESS"
  priority  = 1000

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["35.235.240.0/20"]  # IAP range
  target_tags   = ["allow-iap-ssh"]
}
