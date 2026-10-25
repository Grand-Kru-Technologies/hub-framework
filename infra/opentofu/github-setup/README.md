# GitHub repository setup (environments and variables)

OpenTofu stack for GitHub environments, IAM role variables, and staging encryption secrets.

Requires an admin GitHub personal access token in Doppler as `GITHUB_ADMIN_TOKEN`. Not applied from CI.

```sh
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh github-setup plan
doppler run --project hub-framework --config staging -- \
  ./scripts/setup-staging-opentofu.sh github-setup apply
```

Default apply reviewers: `mnyon-grandkru` with `prevent_self_review = false` (Mark can self-approve until a second reviewer is added).

Plan environment `staging-opentofu-plan` and apply environment `staging-opentofu` both restrict deployments to branch `main`.
