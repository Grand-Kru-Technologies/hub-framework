# Bootstrap (AWS remote state and OIDC roles)

One-time stack that provisions:

- Versioned S3 bucket with SSE-S3, TLS-only bucket policy, lifecycle on noncurrent versions, and `prevent_destroy`
- Read-only IAM role for plan jobs (exact state read plus lock object write/delete only)
- Write IAM role trusted only from the `staging-opentofu` environment on `main`

Uses the existing account GitHub OIDC provider (`data.aws_iam_openid_connect_provider`). The setup script creates the provider when it is missing.

Local backend state is encrypted with `LOCAL_STATE_ENCRYPTION_PASSPHRASE` (separate from staging remote state encryption).

Prefer the repository setup script instead of running this directory in isolation:

```sh
./scripts/setup-staging-opentofu.sh
```
