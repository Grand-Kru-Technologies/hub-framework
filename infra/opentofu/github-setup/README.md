# GitHub repository setup (environments and variables)

Defines repository configuration for staging OpenTofu:

- Environment `staging-opentofu-plan` for plan jobs on `main`
- Environment `staging-opentofu` on `main` only (custom branch policy), with configurable required reviewers
- Repository variables for plan and apply IAM role ARNs
- Environment secrets for staging remote state encryption on both environments

Requires an admin GitHub personal access token (`GITHUB_ADMIN_TOKEN`). Not applied from CI.

Default reviewers: `mnyon-grandkru` with `prevent_self_review = false` so Mark can self-approve until a second reviewer is added.

Local backend state is encrypted with the same `LOCAL_STATE_ENCRYPTION_PASSPHRASE` used for bootstrap.

Run via:

```sh
./scripts/setup-staging-opentofu.sh
```

Railway and Sentry credentials remain manual repository secrets (`RAILWAY_TOKEN`, `STAGING_SENTRY_DSN`).

Backup the generated staging encryption passphrase in Doppler (hub-framework staging config).
