variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region for the remote state bucket and IAM role."
}

variable "state_bucket_name" {
  type        = string
  default     = "qg-hub-staging-tofu-state"
  description = "S3 bucket name for OpenTofu remote state (staging stack)."
}

variable "github_repository" {
  type        = string
  default     = "gk-queer-global/hub-framework"
  description = "GitHub repository allowed to assume the OpenTofu state role (org/repo)."
}

variable "github_oidc_thumbprint" {
  type        = string
  default     = "6938fd4e787b04b6bd9bf7d0bcfd543250d3123f"
  description = "Thumbprint for token.actions.githubusercontent.com (rotate if GitHub rotates their cert)."
}

variable "tofu_state_role_name" {
  type        = string
  default     = "hub-framework-staging-tofu-state"
  description = "IAM role name assumed by GitHub Actions for staging OpenTofu."
}
