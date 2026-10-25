variable "github_owner" {
  type        = string
  default     = "gk-queer-global"
  description = "GitHub organization or user that owns the repository."
}

variable "github_repository" {
  type        = string
  default     = "hub-framework"
  description = "Repository name (without owner)."
}

variable "github_token" {
  type        = string
  sensitive   = true
  description = "PAT or admin token with repo and environment permissions (operators only, not used in CI)."
}

variable "apply_reviewer_username" {
  type        = string
  default     = "mnyon-grandkru"
  description = "Required reviewer for staging-opentofu apply approvals."
}

variable "tofu_plan_role_arn" {
  type        = string
  description = "Read-only plan role ARN from bootstrap output github_tofu_plan_role_arn."
}

variable "tofu_apply_role_arn" {
  type        = string
  description = "Write apply role ARN from bootstrap output github_tofu_apply_role_arn."
}

variable "state_encryption_passphrase" {
  type        = string
  sensitive   = true
  description = "Passphrase from bootstrap output state_encryption_passphrase."
}

variable "plan_environment_name" {
  type        = string
  default     = "staging-opentofu-plan"
  description = "GitHub Environment for read-only plan jobs."
}

variable "apply_environment_name" {
  type        = string
  default     = "staging-opentofu"
  description = "GitHub Environment for apply jobs on main."
}
