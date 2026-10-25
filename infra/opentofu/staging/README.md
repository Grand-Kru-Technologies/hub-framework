# Staging OpenTofu (Railway variables)

Declares staging Railway service variables for hub-app:

- `SENTRY_DSN` (remote forward error reporting)

State is stored in the S3 backend in `versions.tf` with **SSE-S3** and **OpenTofu client-side encryption** (passphrase via `TF_VAR_state_encryption_passphrase` / `TOFU_STATE_ENCRYPTION_PASSPHRASE` secret).

## Bootstrap

Create the state bucket and GitHub OIDC role once via `infra/opentofu/bootstrap/` (see that README).

## CI

Pull requests run `tofu plan` and upload a saved plan artifact via `.github/workflows/staging-opentofu.yml`.

Apply runs only on **`main`** through the **`staging-opentofu`** GitHub Environment (required reviewer), applying a **saved plan artifact** from an approved PR plan run (no fresh `-auto-approve` plan on apply).

Required configuration:

- Secrets: `RAILWAY_TOKEN`, `STAGING_SENTRY_DSN`, `TOFU_STATE_ENCRYPTION_PASSPHRASE`
- Variable: `TOFU_STATE_AWS_ROLE_ARN` (from bootstrap output)

## Import existing SENTRY_DSN

Import id format: `service_id:environment_name:VAR_NAME` (e.g. `…:staging:SENTRY_DSN`).

## Local apply (operators only)

```sh
cd infra/opentofu/staging
export TF_VAR_railway_token="$RAILWAY_TOKEN"
export TF_VAR_sentry_dsn="$STAGING_SENTRY_DSN"
export TF_VAR_state_encryption_passphrase="$TOFU_STATE_ENCRYPTION_PASSPHRASE"
# AWS via OIDC or operator role with S3 access to the state bucket
tofu init
tofu plan
tofu apply
```

Redeploy hub-app after apply so new variables load.
