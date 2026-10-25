# GitHub Actions OIDC thumbprints (published):
# https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services
# AWS IAM requires a thumbprint when registering an OIDC provider; for GitHub's endpoint, AWS does not
# rely on thumbprint validation for ongoing authentication (see AWS IAM OIDC provider documentation).
# Comma-separated values: github-oidc-thumbprints (same file used by scripts/setup-staging-opentofu.sh).

locals {
  github_oidc_thumbprint_list = [
    for t in split(",", trimspace(file("${path.module}/github-oidc-thumbprints"))) : trimspace(t)
  ]
  github_oidc_provider_arn = var.create_github_oidc_provider ? aws_iam_openid_connect_provider.github[0].arn : data.aws_iam_openid_connect_provider.github[0].arn
}

resource "aws_iam_openid_connect_provider" "github" {
  count = var.create_github_oidc_provider ? 1 : 0

  url             = var.github_oidc_url
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = local.github_oidc_thumbprint_list

  lifecycle {
    prevent_destroy = true
  }
}

data "aws_iam_openid_connect_provider" "github" {
  count = var.create_github_oidc_provider ? 0 : 1
  url   = var.github_oidc_url
}
