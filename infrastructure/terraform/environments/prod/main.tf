module "context" {
  source = "../../modules/environment-context"

  project     = var.project
  environment = var.environment
  owner       = var.owner
}

# Add provider-specific modules here once the target is selected:
# - network
# - cluster
# - database
# - artifact storage
# - container registry
#
# Application Deployments, Services, and Namespaces remain owned by deploy/.
