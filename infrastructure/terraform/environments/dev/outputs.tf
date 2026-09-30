output "name_prefix" {
  description = "Prefix to use for this environment's infrastructure."
  value       = module.context.name_prefix
}

output "common_tags" {
  description = "Common metadata for this environment."
  value       = module.context.common_tags
}
