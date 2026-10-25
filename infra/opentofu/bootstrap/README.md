# Bootstrap (AWS remote state and OIDC roles)

OpenTofu stack for the GitHub OIDC provider (when `create_github_oidc_provider = true`, the default), S3 remote state, IAM plan/apply roles, and TLS-only bucket policy.

Run through the repository setup script (recommended):

```sh
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap plan
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap apply
```

On a fresh AWS account with no GitHub OIDC provider, leave `create_github_oidc_provider` at the default so the provider is created in the bootstrap plan you approve. Set `create_github_oidc_provider = false` only when the provider already exists and you want a data lookup instead.

If OpenTofu cannot manage OIDC, use the documented fallback (confirmed): `./scripts/setup-staging-opentofu.sh oidc ensure`.

Local backend state is encrypted; the passphrase lives in Doppler as `LOCAL_STATE_ENCRYPTION_PASSPHRASE`.
