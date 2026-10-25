# Staging OpenTofu (Railway variables)

Declares staging Railway service variables for hub-app:

- `SENTRY_DSN` (remote forward error reporting)

State uses the S3 backend in `versions.tf` with SSE-S3 and client-side encryption.

Pull requests run offline `tofu validate` for bootstrap, github-setup, and staging (no Railway or AWS credentials).

Plan and apply run on `main` only under GitHub environments, using IAM roles from bootstrap.

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
