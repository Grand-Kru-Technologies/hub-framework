# OpenTofu bootstrap (remote state bucket + GitHub OIDC)

One-time (or rare) stack that creates:

- Encrypted, versioned S3 bucket `qg-hub-staging-tofu-state` (SSE-S3)
- GitHub Actions OIDC IAM role with read/write access to that bucket

Uses a **local** backend (`terraform.tfstate` in this directory). Operators apply with administrative AWS credentials; CI never uses long-lived AWS access keys for the staging stack.

After apply, set repository variable **`TOFU_STATE_AWS_ROLE_ARN`** to the `github_tofu_state_role_arn` output.

```sh
cd infra/opentofu/bootstrap
tofu init
tofu plan
tofu apply
```

Then configure GitHub Environment **`staging-opentofu`** with required reviewers before using the apply job in `.github/workflows/staging-opentofu.yml`.
