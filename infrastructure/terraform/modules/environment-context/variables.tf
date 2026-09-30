variable "project" {
  description = "Project identifier used in infrastructure naming."
  type        = string
  nullable    = false

  validation {
    condition = (
      length(var.project) <= 32 &&
      can(regex("^[a-z][a-z0-9]*(-[a-z0-9]+)*$", var.project))
    )
    error_message = "project must be a lowercase identifier of at most 32 characters."
  }
}

variable "environment" {
  description = "Deployment environment."
  type        = string
  nullable    = false

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "environment must be dev, staging, or prod."
  }
}

variable "owner" {
  description = "Team responsible for the infrastructure."
  type        = string
  nullable    = false

  validation {
    condition     = length(trimspace(var.owner)) > 0
    error_message = "owner must not be empty."
  }
}

