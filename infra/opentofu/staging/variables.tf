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

variable "python_handler_url" {
  type        = string
  description = "Internal URL hub-app uses to reach python-handler on staging."
  default     = "http://python-handler.railway.internal:5000"
}

variable "railway_environment_id" {
  type        = string
  default     = "6fbd88c0-89a3-41ff-a0a5-d38b8beda3b2"
  description = "Railway environment ID for staging."
}

variable "hub_app_service_id" {
  type        = string
  default     = "c1129f35-8c17-4516-a5fb-c5ddd2b1b982"
  description = "Railway service ID for hub-app."
}
