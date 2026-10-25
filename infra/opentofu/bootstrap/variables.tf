variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region for the remote state bucket and IAM roles."
}

variable "state_bucket_name" {
  type        = string
  default     = "qg-hub-staging-tofu-state"
  description = "S3 bucket name for OpenTofu remote state (staging stack)."
}

variable "github_repository" {
  type        = string
  default     = "gk-queer-global/hub-framework"
  description = "GitHub repository allowed to assume the OpenTofu state roles (org/repo)."
}

variable "github_plan_environment_name" {
  type        = string
  default     = "staging-opentofu-plan"
  description = "GitHub Environment name used by read-only plan jobs."
}

variable "github_apply_environment_name" {
  type        = string
  default     = "staging-opentofu"
  description = "GitHub Environment name used by apply jobs on main."
}

variable "tofu_plan_role_name" {
  type        = string
  default     = "hub-framework-staging-tofu-plan"
  description = "IAM role name for read-only plan access."
}

variable "tofu_apply_role_name" {
  type        = string
  default     = "hub-framework-staging-tofu-apply"
  description = "IAM role name for apply access."
}
