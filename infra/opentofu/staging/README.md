# Staging OpenTofu (Railway variables)

Declares staging Railway service variables for hub-app:

- `SENTRY_DSN` (remote forward error reporting)

State uses the S3 backend in `versions.tf` with SSE-S3 and client-side encryption (passphrase via environment secret `TOFU_STATE_ENCRYPTION_PASSPHRASE`).

## Bootstrap and GitHub setup

1. Apply `infra/opentofu/bootstrap/` once with admin AWS credentials.
2. Apply `infra/opentofu/github-setup/` once with an admin GitHub token (not run from CI).

## CI

Pull requests and `main` pushes run `tofu plan` under environment `staging-opentofu-plan` and upload `staging-tofu-plan`.

Apply runs only on `main` through environment `staging-opentofu`, after verifying the plan run commit is on `main`, using the saved plan artifact.

Repository secrets still required manually: `RAILWAY_TOKEN`, `STAGING_SENTRY_DSN`.

Variables from GitHub setup: `TOFU_STATE_AWS_PLAN_ROLE_ARN`, `TOFU_STATE_AWS_APPLY_ROLE_ARN`.

## Import existing SENTRY_DSN

Import id format: `service_id:environment_name:VAR_NAME` (example suffix `:staging:SENTRY_DSN`).

## Local apply (operators only)

```sh
cd infra/opentofu/staging
export TF_VAR_railway_token="$RAILWAY_TOKEN"
export TF_VAR_sentry_dsn="$STAGING_SENTRY_DSN"
export TF_VAR_state_encryption_passphrase="$TOFU_STATE_ENCRYPTION_PASSPHRASE"
tofu init
tofu plan
tofu apply
```

Redeploy hub-app after apply so new variables load.
