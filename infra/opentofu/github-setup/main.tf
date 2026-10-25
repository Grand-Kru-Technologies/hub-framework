data "github_user" "apply_reviewer" {
  for_each = toset(var.apply_reviewer_usernames)
  username = each.key
}

resource "github_repository_environment" "plan" {
  repository  = var.github_repository
  environment = var.plan_environment_name
}

resource "github_repository_environment" "apply" {
  repository          = var.github_repository
  environment         = var.apply_environment_name
  prevent_self_review = var.prevent_self_review
  wait_timer          = 0

  reviewers {
    users = [for name in var.apply_reviewer_usernames : data.github_user.apply_reviewer[name].id]
  }

  deployment_branch_policy {
    protected_branches     = false
    custom_branch_policies = true
  }
}

resource "github_repository_environment_deployment_policy" "apply_main" {
  repository     = var.github_repository
  environment    = github_repository_environment.apply.environment
  branch_pattern = "main"
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

resource "github_actions_environment_secret" "state_encryption_passphrase_plan" {
  repository      = var.github_repository
  environment     = github_repository_environment.plan.environment
  secret_name     = "TOFU_STATE_ENCRYPTION_PASSPHRASE"
  plaintext_value = var.state_encryption_passphrase
}

resource "github_actions_environment_secret" "state_encryption_passphrase_apply" {
  repository      = var.github_repository
  environment     = github_repository_environment.apply.environment
  secret_name     = "TOFU_STATE_ENCRYPTION_PASSPHRASE"
  plaintext_value = var.state_encryption_passphrase
}
