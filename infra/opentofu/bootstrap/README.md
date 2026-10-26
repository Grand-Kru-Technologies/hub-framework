# Bootstrap (AWS remote state and OIDC roles)

OpenTofu stack for the GitHub OIDC provider (when `create_github_oidc_provider = true`), S3 remote state, IAM plan/apply roles, and TLS-only bucket policy.

Run through the repository setup script (recommended):

```sh
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap plan
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap apply
```

Before `bootstrap plan`, `all plan`, or `all setup`, the setup script runs `tofu init`, then checks bootstrap state for `aws_iam_openid_connect_provider.github`, then runs a read-only `aws iam list-open-id-connect-providers` call. If the AWS call fails (credentials, permissions, throttling), the script stops before `tofu plan`. If bootstrap state already tracks the provider, it keeps `create_github_oidc_provider=true`. Otherwise, if a `token.actions.githubusercontent.com` provider already exists in AWS, it sets `create_github_oidc_provider=false`; if not, it sets `true`. The script logs the choice. Override with `--create-github-oidc-provider true` or `--create-github-oidc-provider false` when you need to force create vs lookup mode.

On a fresh AWS account with no GitHub OIDC provider, detection chooses create mode so the provider appears in the bootstrap plan you approve. If OpenTofu cannot manage OIDC, use the documented fallback (confirmed): `./scripts/setup-staging-opentofu.sh oidc ensure`.

OIDC TLS thumbprints are listed in `github-oidc-thumbprints` (see the comment in `oidc.tf` for GitHub and AWS documentation links).

Local backend state is encrypted; the passphrase lives in Doppler as `LOCAL_STATE_ENCRYPTION_PASSPHRASE`.
