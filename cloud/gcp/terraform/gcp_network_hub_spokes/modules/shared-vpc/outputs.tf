output "host_project_id" {
  description = "The shared VPC host project ID"
  value       = google_compute_shared_vpc_host_project.host.project
}

output "service_project_ids" {
  description = "List of attached service project IDs"
  value       = [for sp in google_compute_shared_vpc_service_project.service : sp.service_project]
}
