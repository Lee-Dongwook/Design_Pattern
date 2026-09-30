variable "project" {
  description = "Project identifier."
  type        = string
  nullable    = false
}

variable "environment" {
  description = "Deployment environment."
  type        = string
  nullable    = false
}

variable "owner" {
  description = "Infrastructure owner."
  type        = string
  nullable    = false
}
