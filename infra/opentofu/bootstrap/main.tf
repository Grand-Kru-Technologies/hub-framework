data "aws_caller_identity" "current" {}

resource "aws_s3_bucket" "tofu_state" {
  bucket = var.state_bucket_name
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

resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [var.github_oidc_thumbprint]
}

data "aws_iam_policy_document" "github_assume_role" {
  statement {
    effect = "Allow"
    actions = [
      "sts:AssumeRoleWithWebIdentity",
    ]
    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
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
        "repo:${var.github_repository}:ref:refs/heads/main",
        "repo:${var.github_repository}:pull_request",
        "repo:${var.github_repository}:environment:staging-opentofu",
      ]
    }
  }
}

resource "aws_iam_role" "github_tofu_state" {
  name               = var.tofu_state_role_name
  assume_role_policy = data.aws_iam_policy_document.github_assume_role.json
}

data "aws_iam_policy_document" "tofu_state_s3" {
  statement {
    sid = "ListStateBucket"
    actions = [
      "s3:ListBucket",
    ]
    resources = [aws_s3_bucket.tofu_state.arn]
  }

  statement {
    sid = "ReadWriteStateObjects"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = ["${aws_s3_bucket.tofu_state.arn}/*"]
  }
}

resource "aws_iam_role_policy" "github_tofu_state_s3" {
  name   = "tofu-state-s3"
  role   = aws_iam_role.github_tofu_state.id
  policy = data.aws_iam_policy_document.tofu_state_s3.json
}

output "state_bucket_name" {
  value       = aws_s3_bucket.tofu_state.id
  description = "S3 bucket for staging OpenTofu remote state."
}

output "github_tofu_state_role_arn" {
  value       = aws_iam_role.github_tofu_state.arn
  description = "IAM role ARN for GitHub Actions (configure as repository variable TOFU_STATE_AWS_ROLE_ARN)."
}

output "aws_account_id" {
  value       = data.aws_caller_identity.current.account_id
  description = "AWS account hosting remote state."
}
