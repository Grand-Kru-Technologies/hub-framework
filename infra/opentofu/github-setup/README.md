# GitHub repository setup (environments and variables)

Defines repository configuration for staging OpenTofu:

- Environment `staging-opentofu-plan` for read-only plan jobs
- Environment `staging-opentofu` on `main` with required reviewer, no self-review
- Repository variables for plan and apply IAM role ARNs
- Environment secrets for the state encryption passphrase on both environments

Not applied from CI. Operators run once after bootstrap apply:

```sh
cd infra/opentofu/github-setup
export TF_VAR_github_token="$GITHUB_ADMIN_TOKEN"
export TF_VAR_tofu_plan_role_arn="<from bootstrap output>"
export TF_VAR_tofu_apply_role_arn="<from bootstrap output>"
export TF_VAR_state_encryption_passphrase="<from bootstrap output>"
tofu init
tofu plan
tofu apply
```

Railway and Sentry credentials remain manual repository secrets (`RAILWAY_TOKEN`, `STAGING_SENTRY_DSN`).
