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

variable "staging_state_key" {
  type        = string
  default     = "hub-framework/staging/terraform.tfstate"
  description = "S3 object key for staging remote state (must match staging/versions.tf backend key)."
}

variable "local_state_encryption_passphrase" {
  type        = string
  sensitive   = true
  description = "Passphrase for encrypting this stack's local terraform.tfstate (operator-supplied; not the staging remote state secret)."
}

variable "create_github_oidc_provider" {
  type        = bool
  default     = true
  description = "When true, create the GitHub Actions OIDC provider in this account. When false, look up an existing provider (plan fails if it is missing)."
}

variable "github_oidc_url" {
  type        = string
  default     = "https://token.actions.githubusercontent.com"
  description = "GitHub Actions OIDC issuer URL."
}

variable "github_oidc_thumbprint" {
  type        = string
  default     = "6938fd4d98bab03fa76916958d6cb6a0a5769a08"
  description = "SHA-1 thumbprint for the GitHub OIDC TLS certificate (40 hex characters)."
}
