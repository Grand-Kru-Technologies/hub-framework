# Staging OpenTofu (Railway variables)

This stack declares staging `SENTRY_DSN` for hub-app so error reporting is managed as code instead of a manual Railway dashboard edit.

## Apply (not run from CI by default)

From this directory:

```sh
export TF_VAR_railway_token="$RAILWAY_TOKEN"
export TF_VAR_sentry_dsn="$SENTRY_DSN"
tofu init
tofu plan
tofu apply
```

Requirements:

- OpenTofu or Terraform 1.6+
- A Railway project token with access to `qg-hub-staging` / staging
- The Sentry DSN value from your Sentry project (store in your team secret manager, not in git)

After apply, redeploy hub-app so the new variable is picked up (for example via `.github/workflows/deploy.yml` once Railway credentials are configured on the repo).
