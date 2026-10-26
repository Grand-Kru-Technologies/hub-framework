provider "railway" {
  token = var.railway_token
}

resource "railway_variable" "hub_app_sentry_dsn" {
  environment_id = var.railway_environment_id
  service_id     = var.hub_app_service_id
  name           = "SENTRY_DSN"
  value          = var.sentry_dsn
}

import {
  to = railway_variable.hub_app_sentry_dsn
  id = "${var.hub_app_service_id}:${var.railway_environment_name}:SENTRY_DSN"
}
