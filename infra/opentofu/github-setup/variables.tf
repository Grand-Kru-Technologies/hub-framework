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
  description = "Admin personal access token with repo and environment permissions (operators only, not used in CI)."
}

variable "apply_reviewer_usernames" {
  type        = list(string)
  default     = ["mnyon-grandkru"]
  description = "Required reviewers for staging-opentofu apply approvals. Add a second username if Mark wants to avoid self-review deadlock when prevent_self_review is true."
}

variable "prevent_self_review" {
  type        = bool
  default     = false
  description = "When true, the user who triggered apply cannot approve their own deployment. Default false so Mark can self-approve until a second reviewer is added."
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
  description = "Staging remote state encryption passphrase (generated outside Terraform; store backup in Doppler hub-framework staging config)."
}

variable "plan_environment_name" {
  type        = string
  default     = "staging-opentofu-plan"
  description = "GitHub Environment for plan jobs on main."
}

variable "apply_environment_name" {
  type        = string
  default     = "staging-opentofu"
  description = "GitHub Environment for apply jobs on main."
}

variable "local_state_encryption_passphrase" {
  type        = string
  sensitive   = true
  description = "Passphrase for encrypting this stack's local terraform.tfstate (operator-supplied)."
}
