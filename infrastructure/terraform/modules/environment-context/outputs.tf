output "name_prefix" {
  description = "Common prefix for infrastructure resource names."
  value       = local.name_prefix
}

output "common_tags" {
  description = "Provider-neutral ownership and environment metadata."
  value       = local.common_tags
}
