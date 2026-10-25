# Staging OpenTofu (Railway variables)

Declares staging Railway service variables for hub-app:

- `SENTRY_DSN` (optional error reporting for remote forwarders)
- `PYTHON_HANDLER_URL` (internal python-handler URL)

State is stored in the encrypted S3 backend configured in `versions.tf`.

## CI

Pull requests that touch this directory run `tofu plan` via `.github/workflows/staging-opentofu.yml`.

Apply runs only from a manual workflow dispatch when the `confirm_apply` input is exactly `apply`.

Required repository secrets for plan/apply:

- `RAILWAY_TOKEN`
- `STAGING_SENTRY_DSN`
- `TOFU_STATE_AWS_ACCESS_KEY_ID`
- `TOFU_STATE_AWS_SECRET_ACCESS_KEY`

The state bucket `qg-hub-staging-tofu-state` must exist with encryption enabled before the first init.

## Import existing SENTRY_DSN

`main.tf` includes an import block for an existing `SENTRY_DSN` variable. If Railway uses a different import id format, adjust the import `id` after `tofu plan` reports the expected value.

## Local apply (operators only)

```sh
cd infra/opentofu/staging
export TF_VAR_railway_token="$RAILWAY_TOKEN"
export TF_VAR_sentry_dsn="$SENTRY_DSN"
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
tofu init
tofu plan
tofu apply
```

Redeploy hub-app after apply so new variables load.
