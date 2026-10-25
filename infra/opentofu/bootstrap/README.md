# Bootstrap (AWS remote state and OIDC roles)

OpenTofu stack for the GitHub OIDC provider (when `create_github_oidc_provider = true`), S3 remote state, IAM plan/apply roles, and TLS-only bucket policy.

Run through the repository setup script (recommended):

```sh
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap plan
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap apply
```

Before `bootstrap plan`, `all plan`, or `all setup`, the setup script runs a read-only `aws iam list-open-id-connect-providers` check. If a `token.actions.githubusercontent.com` provider already exists, it sets `create_github_oidc_provider=false` for that plan; otherwise it sets `true` and logs the choice. Override with `--create-github-oidc-provider true` or `--create-github-oidc-provider false` when you need to force create vs lookup mode.

On a fresh AWS account with no GitHub OIDC provider, detection chooses create mode so the provider appears in the bootstrap plan you approve. If OpenTofu cannot manage OIDC, use the documented fallback (confirmed): `./scripts/setup-staging-opentofu.sh oidc ensure`.

OIDC TLS thumbprints are listed in `github-oidc-thumbprints` (see the comment in `oidc.tf` for GitHub and AWS documentation links).

Local backend state is encrypted; the passphrase lives in Doppler as `LOCAL_STATE_ENCRYPTION_PASSPHRASE`.
