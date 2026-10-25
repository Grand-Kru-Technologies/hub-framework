#!/usr/bin/env bash
# One-time staging OpenTofu setup: AWS bootstrap, GitHub environments/variables/secrets.
# Does not print secrets. Staging encryption passphrase must be backed up (see README).
#
# Usage: setup-staging-opentofu.sh [--push-baseline-tag] [--dry-run]
#
# Required env:
#   AWS credentials with permission to manage S3/IAM (bootstrap)
#   GITHUB_ADMIN_TOKEN  admin PAT for github-setup (repo + environments)
#   LOCAL_STATE_ENCRYPTION_PASSPHRASE  encrypts bootstrap/github-setup local tfstate
#
# Optional env:
#   STAGING_STATE_ENCRYPTION_PASSPHRASE  if unset, a new value is generated to a temp file
#   BASELINE_TAG_COMMIT  default 1583326b for hub-framework-staging-baseline
#
# shellcheck disable=SC2317
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BOOTSTRAP_DIR="$ROOT/infra/opentofu/bootstrap"
GITHUB_DIR="$ROOT/infra/opentofu/github-setup"
GITHUB_OIDC_URL="https://token.actions.githubusercontent.com"
GITHUB_OIDC_THUMBPRINT="6938fd4d98bab03fa76833955d3977e6238c545d"

PUSH_BASELINE_TAG=0
DRY_RUN=0

usage() {
  cat <<EOF >&2
usage: $(basename "$0") [--push-baseline-tag] [--dry-run]

  --push-baseline-tag  Create and push hub-framework-staging-baseline (once)
  --dry-run            Validate arguments and environment only
EOF
  exit 64
}

while [ $# -gt 0 ]; do
  case "$1" in
    --push-baseline-tag) PUSH_BASELINE_TAG=1 ;;
    --dry-run) DRY_RUN=1 ;;
    -h|--help) usage ;;
    *) echo "unknown argument: $1" >&2; usage ;;
  esac
  shift
done

require_env() {
  local name=$1
  if [ -z "${!name:-}" ]; then
    echo "$name is required" >&2
    exit 1
  fi
}

require_env LOCAL_STATE_ENCRYPTION_PASSPHRASE
require_env GITHUB_ADMIN_TOKEN

if [ "$DRY_RUN" -eq 1 ]; then
  echo "dry-run ok"
  exit 0
fi

ensure_github_oidc_provider() {
  if aws iam list-open-id-connect-providers --output json \
    | jq -e '.OpenIDConnectProviderList[] | select(.Arn | contains("token.actions.githubusercontent.com"))' >/dev/null; then
    echo "GitHub OIDC provider already exists"
    return 0
  fi
  echo "Creating GitHub OIDC provider"
  aws iam create-open-id-connect-provider \
    --url "$GITHUB_OIDC_URL" \
    --client-id-list sts.amazonaws.com \
    --thumbprint-list "$GITHUB_OIDC_THUMBPRINT" >/dev/null
}

TMPDIR="$(mktemp -d)"
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT
chmod 700 "$TMPDIR"

STAGING_PASS_FILE="$TMPDIR/staging-passphrase"
if [ -n "${STAGING_STATE_ENCRYPTION_PASSPHRASE:-}" ]; then
  printf '%s' "$STAGING_STATE_ENCRYPTION_PASSPHRASE" >"$STAGING_PASS_FILE"
else
  openssl rand -base64 32 >"$STAGING_PASS_FILE"
fi
chmod 600 "$STAGING_PASS_FILE"

ensure_github_oidc_provider

export TF_VAR_local_state_encryption_passphrase="$LOCAL_STATE_ENCRYPTION_PASSPHRASE"

echo "Applying bootstrap (AWS)"
(
  cd "$BOOTSTRAP_DIR"
  tofu init -input=false
  tofu apply -input=false -auto-approve
  tofu output -raw github_tofu_plan_role_arn >"$TMPDIR/plan_role_arn"
  tofu output -raw github_tofu_apply_role_arn >"$TMPDIR/apply_role_arn"
)

export TF_VAR_github_token="$GITHUB_ADMIN_TOKEN"
export TF_VAR_tofu_plan_role_arn
TF_VAR_tofu_plan_role_arn="$(cat "$TMPDIR/plan_role_arn")"
export TF_VAR_tofu_apply_role_arn
TF_VAR_tofu_apply_role_arn="$(cat "$TMPDIR/apply_role_arn")"
export TF_VAR_state_encryption_passphrase
TF_VAR_state_encryption_passphrase="$(cat "$STAGING_PASS_FILE")"

echo "Applying github-setup"
(
  cd "$GITHUB_DIR"
  tofu init -input=false
  tofu apply -input=false -auto-approve
)

echo "Store the staging state encryption passphrase in Doppler (hub-framework staging config)."
echo "Passphrase file (delete after backup): $STAGING_PASS_FILE"

if [ "$PUSH_BASELINE_TAG" -eq 1 ]; then
  BASELINE_TAG_COMMIT="${BASELINE_TAG_COMMIT:-1583326b}"
  if git ls-remote --exit-code --tags origin refs/tags/hub-framework-staging-baseline >/dev/null 2>&1; then
    echo "hub-framework-staging-baseline already exists on origin"
  else
    git tag -a hub-framework-staging-baseline -m "Pre remote-forward-fix baseline (main ${BASELINE_TAG_COMMIT})" "$BASELINE_TAG_COMMIT"
    git push origin hub-framework-staging-baseline
  fi
fi
