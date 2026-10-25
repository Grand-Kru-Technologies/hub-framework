variable "railway_token" {
  type        = string
  sensitive   = true
  description = "Railway project token for qg-hub-staging (staging environment)."
}

variable "sentry_dsn" {
  type        = string
  sensitive   = true
  description = "Sentry DSN for hub-app remote forward error reporting on staging."
}

variable "state_encryption_passphrase" {
  type        = string
  sensitive   = true
  description = "Passphrase for OpenTofu client-side state/plan encryption (see versions.tf encryption block)."
}

variable "railway_environment_id" {
  type        = string
  default     = "6fbd88c0-89a3-41ff-a0a5-d38b8beda3b2"
  description = "Railway environment ID for staging."
}

variable "railway_environment_name" {
  type        = string
  default     = "staging"
  description = "Railway environment name (used in import IDs: service_id:environment_name:VAR_NAME)."
}

variable "hub_app_service_id" {
  type        = string
  default     = "c1129f35-8c17-4516-a5fb-c5ddd2b1b982"
  description = "Railway service ID for hub-app."
}
