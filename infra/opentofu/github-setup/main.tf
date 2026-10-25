data "github_user" "apply_reviewer" {
  username = var.apply_reviewer_username
}

resource "github_repository_environment" "plan" {
  repository  = var.github_repository
  environment = var.plan_environment_name
}

resource "github_repository_environment" "apply" {
  repository           = var.github_repository
  environment          = var.apply_environment_name
  prevent_self_review  = true
  wait_timer           = 0

  reviewers {
    users = [data.github_user.apply_reviewer.id]
  }

  deployment_branch_policy {
    protected_branches     = true
    custom_branch_policies = false
  }
}

resource "github_actions_variable" "tofu_plan_role_arn" {
  repository    = var.github_repository
  variable_name = "TOFU_STATE_AWS_PLAN_ROLE_ARN"
  value         = var.tofu_plan_role_arn
}

resource "github_actions_variable" "tofu_apply_role_arn" {
  repository    = var.github_repository
  variable_name = "TOFU_STATE_AWS_APPLY_ROLE_ARN"
  value         = var.tofu_apply_role_arn
}

resource "github_actions_environment_secret" "state_encryption_passphrase" {
  repository      = var.github_repository
  environment     = var.plan_environment_name
  secret_name     = "TOFU_STATE_ENCRYPTION_PASSPHRASE"
  plaintext_value = var.state_encryption_passphrase
}

resource "github_actions_environment_secret" "state_encryption_passphrase_apply" {
  repository      = var.github_repository
  environment     = var.apply_environment_name
  secret_name     = "TOFU_STATE_ENCRYPTION_PASSPHRASE"
  plaintext_value = var.state_encryption_passphrase
}
