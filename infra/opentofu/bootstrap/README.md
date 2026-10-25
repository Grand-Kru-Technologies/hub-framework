# Bootstrap (AWS remote state and OIDC roles)

One-time stack that provisions:

- Versioned S3 bucket with SSE-S3, TLS-only bucket policy, lifecycle on noncurrent versions, and `prevent_destroy`
- Read-only IAM role for plan jobs (prefix `hub-framework/staging/` only)
- Write IAM role trusted only from the `staging-opentofu` environment on `main`
- Random client-side state encryption passphrase (sensitive output)

Uses the existing account GitHub OIDC provider (`data.aws_iam_openid_connect_provider`). If that provider is missing, create it once in AWS (thumbprint is the 40-character SHA-1 fingerprint for `token.actions.githubusercontent.com`) before running this stack.

Local backend only. Apply with administrative AWS credentials.

```sh
cd infra/opentofu/bootstrap
tofu init
tofu plan
tofu apply
```

Then run the GitHub setup stack in `../github-setup/` (not applied from CI) to create environments, repository variables, and the encryption secret.
