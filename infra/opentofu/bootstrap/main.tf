data "aws_caller_identity" "current" {}

data "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
}

resource "random_password" "state_encryption" {
  length  = 32
  special = true
}

resource "aws_s3_bucket" "tofu_state" {
  bucket = var.state_bucket_name

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_s3_bucket_public_access_block" "tofu_state" {
  bucket = aws_s3_bucket.tofu_state.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "tofu_state" {
  bucket = aws_s3_bucket.tofu_state.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tofu_state" {
  bucket = aws_s3_bucket.tofu_state.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "tofu_state" {
  bucket = aws_s3_bucket.tofu_state.id

  rule {
    id     = "expire-noncurrent-versions"
    status = "Enabled"

    noncurrent_version_expiration {
      noncurrent_days = 90
    }
  }
}

data "aws_iam_policy_document" "deny_insecure_transport" {
  statement {
    sid    = "DenyInsecureTransport"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = ["s3:*"]

    resources = [
      aws_s3_bucket.tofu_state.arn,
      "${aws_s3_bucket.tofu_state.arn}/*",
    ]

    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

resource "aws_s3_bucket_policy" "tofu_state" {
  bucket = aws_s3_bucket.tofu_state.id
  policy = data.aws_iam_policy_document.deny_insecure_transport.json
}

locals {
  state_prefix = "hub-framework/staging/"
  state_objects_arn = "${aws_s3_bucket.tofu_state.arn}/${local.state_prefix}*"
}

data "aws_iam_policy_document" "github_plan_assume_role" {
  statement {
    effect = "Allow"
    actions = [
      "sts:AssumeRoleWithWebIdentity",
    ]
    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github.arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${var.github_repository}:pull_request",
        "repo:${var.github_repository}:ref:refs/heads/main",
        "repo:${var.github_repository}:environment:${var.github_plan_environment_name}",
      ]
    }
  }
}

resource "aws_iam_role" "github_tofu_plan" {
  name               = var.tofu_plan_role_name
  assume_role_policy = data.aws_iam_policy_document.github_plan_assume_role.json
}

data "aws_iam_policy_document" "tofu_state_plan_s3" {
  statement {
    sid = "ListStagingStatePrefix"
    actions = [
      "s3:ListBucket",
    ]
    resources = [aws_s3_bucket.tofu_state.arn]
    condition {
      test     = "StringLike"
      variable = "s3:prefix"
      values   = ["${local.state_prefix}*"]
    }
  }

  statement {
    sid = "ReadStagingStateObjects"
    actions = [
      "s3:GetObject",
    ]
    resources = [local.state_objects_arn]
  }

  statement {
    sid = "WriteStagingStateLock"
    actions = [
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = ["${aws_s3_bucket.tofu_state.arn}/${local.state_prefix}*.tflock"]
  }
}

resource "aws_iam_role_policy" "github_tofu_plan_s3" {
  name   = "tofu-state-plan-read"
  role   = aws_iam_role.github_tofu_plan.id
  policy = data.aws_iam_policy_document.tofu_state_plan_s3.json
}

data "aws_iam_policy_document" "github_apply_assume_role" {
  statement {
    effect = "Allow"
    actions = [
      "sts:AssumeRoleWithWebIdentity",
    ]
    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github.arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${var.github_repository}:environment:${var.github_apply_environment_name}",
      ]
    }
  }
}

resource "aws_iam_role" "github_tofu_apply" {
  name               = var.tofu_apply_role_name
  assume_role_policy = data.aws_iam_policy_document.github_apply_assume_role.json
}

data "aws_iam_policy_document" "tofu_state_apply_s3" {
  statement {
    sid = "ListStagingStatePrefix"
    actions = [
      "s3:ListBucket",
    ]
    resources = [aws_s3_bucket.tofu_state.arn]
    condition {
      test     = "StringLike"
      variable = "s3:prefix"
      values   = ["${local.state_prefix}*"]
    }
  }

  statement {
    sid = "ReadWriteStagingStateObjects"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = [local.state_objects_arn]
  }
}

resource "aws_iam_role_policy" "github_tofu_apply_s3" {
  name   = "tofu-state-apply-write"
  role   = aws_iam_role.github_tofu_apply.id
  policy = data.aws_iam_policy_document.tofu_state_apply_s3.json
}

output "state_bucket_name" {
  value       = aws_s3_bucket.tofu_state.id
  description = "S3 bucket for staging OpenTofu remote state."
}

output "github_tofu_plan_role_arn" {
  value       = aws_iam_role.github_tofu_plan.arn
  description = "Read-only IAM role for plan jobs (repository variable TOFU_STATE_AWS_PLAN_ROLE_ARN)."
}

output "github_tofu_apply_role_arn" {
  value       = aws_iam_role.github_tofu_apply.arn
  description = "Write IAM role for apply jobs (repository variable TOFU_STATE_AWS_APPLY_ROLE_ARN)."
}

output "state_encryption_passphrase" {
  value       = random_password.state_encryption.result
  sensitive   = true
  description = "Client-side state encryption passphrase (repository secret TOFU_STATE_ENCRYPTION_PASSPHRASE)."
}

output "aws_account_id" {
  value       = data.aws_caller_identity.current.account_id
  description = "AWS account hosting remote state."
}

output "github_oidc_provider_arn" {
  value       = data.aws_iam_openid_connect_provider.github.arn
  description = "Existing GitHub OIDC provider ARN used for trust policies."
}
