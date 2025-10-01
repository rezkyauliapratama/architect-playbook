output "spoke_nonprod_project_id" {
  description = "Spoke non-prod project ID"
  value       = module.spoke_nonprod_project.project_id
}

output "spoke_nonprod_network_self_link" {
  description = "Spoke non-prod network self link"
  value       = module.spoke_nonprod_network.network_self_link
}

output "spoke_nonprod_network_name" {
  description = "Spoke non-prod network name"
  value       = module.spoke_nonprod_network.network_name
}

output "spoke_nonprod_subnets" {
  description = "Spoke non-prod subnets details"
  value       = module.spoke_nonprod_network.subnets
}
