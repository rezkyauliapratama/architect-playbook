output "hub_project_id" {
  description = "Hub project ID"
  value       = module.hub_project.project_id
}

output "hub_network_self_link" {
  description = "Hub network self link"
  value       = module.hub_network.network_self_link
}

output "hub_network_name" {
  description = "Hub network name"
  value       = module.hub_network.network_name
}

output "hub_subnets" {
  description = "Hub subnets details"
  value       = module.hub_network.subnets
}

output "nat_ips" {
  description = "NAT IP addresses"
  value       = module.hub_network.nat_ips
}
