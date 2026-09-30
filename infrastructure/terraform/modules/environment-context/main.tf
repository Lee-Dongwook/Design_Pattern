locals {
  name_prefix = "${var.project}-${var.environment}"

  common_tags = {
    project     = var.project
    environment = var.environment
    owner       = trimspace(var.owner)
    managed_by  = "terraform"
  }
}
