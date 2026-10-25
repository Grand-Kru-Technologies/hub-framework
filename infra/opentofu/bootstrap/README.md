# Bootstrap (AWS remote state and OIDC roles)

OpenTofu stack for S3 remote state, IAM plan/apply roles, and TLS-only bucket policy.

Run through the repository setup script (recommended):

```sh
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap plan
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh bootstrap apply
```

Local backend state is encrypted; the passphrase lives in Doppler as `LOCAL_STATE_ENCRYPTION_PASSPHRASE`.
