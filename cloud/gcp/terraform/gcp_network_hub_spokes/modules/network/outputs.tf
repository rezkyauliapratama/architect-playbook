output "network_self_link" {
  description = "Self link of the VPC network"
  value       = google_compute_network.vpc.self_link
}

output "network_id" {
  description = "ID of the VPC network"
  value       = google_compute_network.vpc.id
}

output "network_name" {
  description = "Name of the VPC network"
  value       = google_compute_network.vpc.name
}

output "subnets" {
  description = "Map of subnet details"
  value = {
    for k, v in google_compute_subnetwork.subnets : k => {
      name          = v.name
      id            = v.id
      self_link     = v.self_link
      ip_cidr_range = v.ip_cidr_range
      region        = v.region
    }
  }
}

output "nat_ips" {
  description = "List of NAT IP addresses"
  value       = var.enable_nat ? google_compute_address.nat_ips[*].address : []
}

output "router_name" {
  description = "Name of the Cloud Router"
  value       = var.enable_nat ? google_compute_router.router[0].name : null
}
