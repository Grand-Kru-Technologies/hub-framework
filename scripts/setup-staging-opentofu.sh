#!/usr/bin/env bash
# Library-style OpenTofu staging setup (bootstrap + github-setup).
# Secrets: use Doppler (doppler run -- ./scripts/setup-staging-opentofu.sh ...) or --doppler-project/--doppler-config.
# Never prints secret values.
#
# Usage:
#   setup-staging-opentofu.sh [options] <command>
#
# Commands:
#   oidc ensure
#   bootstrap plan | apply
#   github-setup plan | apply
#   all plan | apply
#
# Options:
#   --doppler-project NAME   default: hub-framework
#   --doppler-config NAME      default: staging
#   --repo-root PATH           default: repository root (parent of scripts/)
#   --baseline-tag-commit SHA  default: 1583326b
#   --push-baseline-tag        after all apply, push hub-framework-staging-baseline if missing
#   --yes                      skip interactive apply confirmation (unsafe)
#   --dry-run                  validate CLI and arguments only
#
set -euo pipefail

DEFAULT_DOPPLER_PROJECT="hub-framework"
DEFAULT_DOPPLER_CONFIG="staging"
DEFAULT_BASELINE_COMMIT="1583326b"
DEFAULT_BASELINE_TAG="hub-framework-staging-baseline"
GITHUB_OIDC_URL="https://token.actions.githubusercontent.com"
GITHUB_OIDC_THUMBPRINT="6938fd4d98bab03fa76833955d3977e6238c545d"

SECRET_LOCAL_STATE_PASS="LOCAL_STATE_ENCRYPTION_PASSPHRASE"
SECRET_STAGING_STATE_PASS="TOFU_STATE_ENCRYPTION_PASSPHRASE"
SECRET_GITHUB_ADMIN="GITHUB_ADMIN_TOKEN"

DOPPLER_PROJECT="$DEFAULT_DOPPLER_PROJECT"
DOPPLER_CONFIG="$DEFAULT_DOPPLER_CONFIG"
REPO_ROOT=""
BASELINE_TAG_COMMIT="$DEFAULT_BASELINE_COMMIT"
PUSH_BASELINE_TAG=0
AUTO_YES=0
DRY_RUN=0

usage() {
  cat <<EOF >&2
usage: $(basename "$0") [options] <command>

Commands:
  oidc ensure                 Ensure GitHub OIDC provider exists in AWS
  bootstrap plan              Plan bootstrap stack (infra/opentofu/bootstrap)
  bootstrap apply             Apply saved bootstrap plan (bootstrap/tfplan)
  github-setup plan           Plan github-setup (requires bootstrap applied)
  github-setup apply          Apply saved github-setup plan
  all plan                    bootstrap plan then github-setup plan
  all apply                   bootstrap apply then github-setup apply

Options:
  --doppler-project NAME      Doppler project (default: ${DEFAULT_DOPPLER_PROJECT})
  --doppler-config NAME       Doppler config (default: ${DEFAULT_DOPPLER_CONFIG})
  --repo-root PATH            Repository root
  --baseline-tag-commit SHA   Commit for baseline tag (default: ${DEFAULT_BASELINE_COMMIT})
  --push-baseline-tag         Push ${DEFAULT_BASELINE_TAG} when absent on origin
  --yes                       Apply without typing yes (unsafe)
  --dry-run                   Validate options and Doppler CLI only
  -h, --help
EOF
  exit 64
}

log() { printf '%s\n' "$*" >&2; }

doppler_cli() {
  doppler "$@" --project "$DOPPLER_PROJECT" --config "$DOPPLER_CONFIG"
}

doppler_secret_exists() {
  local name=$1
  doppler_cli secrets get "$name" --plain >/dev/null 2>&1
}

doppler_secret_get() {
  local name=$1
  doppler_cli secrets get "$name" --plain
}

doppler_secret_set() {
  local name=$1 value=$2
  printf '%s' "$value" | doppler_cli secrets set "$name" --stdin --no-interactive >/dev/null
}

ensure_doppler_secret() {
  local name=$1 generator=$2
  if doppler_secret_exists "$name"; then
    doppler_secret_get "$name"
    return 0
  fi
  local value
  value="$("$generator")"
  doppler_secret_set "$name" "$value"
  printf '%s' "$value"
}

generate_passphrase() {
  openssl rand -base64 32
}

load_local_state_passphrase() {
  ensure_doppler_secret "$SECRET_LOCAL_STATE_PASS" generate_passphrase
}

load_staging_state_passphrase() {
  ensure_doppler_secret "$SECRET_STAGING_STATE_PASS" generate_passphrase
}

load_github_admin_token() {
  if ! doppler_secret_exists "$SECRET_GITHUB_ADMIN"; then
    log "Doppler secret ${SECRET_GITHUB_ADMIN} is required for github-setup"
    exit 1
  fi
  doppler_secret_get "$SECRET_GITHUB_ADMIN"
}

ensure_github_oidc_provider() {
  if aws iam list-open-id-connect-providers --output json \
    | jq -e '.OpenIDConnectProviderList[] | select(.Arn | contains("token.actions.githubusercontent.com"))' >/dev/null; then
    log "GitHub OIDC provider already exists"
    return 0
  fi
  log "Creating GitHub OIDC provider"
  aws iam create-open-id-connect-provider \
    --url "$GITHUB_OIDC_URL" \
    --client-id-list sts.amazonaws.com \
    --thumbprint-list "$GITHUB_OIDC_THUMBPRINT" >/dev/null
}

confirm_apply() {
  local stack=$1
  if [ "$AUTO_YES" -eq 1 ]; then
    log "warning: applying ${stack} without interactive confirmation (--yes)"
    return 0
  fi
  printf 'Review the plan above. Type yes to apply %s: ' "$stack" >&2
  local ans
  read -r ans
  if [ "$ans" != "yes" ]; then
    log "Apply cancelled (expected yes)"
    exit 1
  fi
}

tofu_init() {
  local dir=$1
  (cd "$dir" && tofu init -input=false)
}

tofu_plan_stack() {
  local dir=$1 out=$2
  shift 2
  (cd "$dir" && tofu plan -input=false -no-color -out="$out" "$@")
}

tofu_show_plan() {
  local dir=$1 plan=$2
  (cd "$dir" && tofu show -no-color "$plan")
}

tofu_apply_plan() {
  local dir=$1 plan=$2
  (cd "$dir" && tofu apply -input=false -no-color "$plan")
}

bootstrap_dir() { printf '%s/infra/opentofu/bootstrap' "$REPO_ROOT"; }
github_setup_dir() { printf '%s/infra/opentofu/github-setup' "$REPO_ROOT"; }

bootstrap_plan() {
  local pass
  pass="$(load_local_state_passphrase)"
  export TF_VAR_local_state_encryption_passphrase="$pass"
  ensure_github_oidc_provider
  local dir
  dir="$(bootstrap_dir)"
  tofu_init "$dir"
  tofu_plan_stack "$dir" tfplan
  log "Bootstrap plan written to ${dir}/tfplan"
}

bootstrap_apply() {
  local pass dir
  pass="$(load_local_state_passphrase)"
  export TF_VAR_local_state_encryption_passphrase="$pass"
  dir="$(bootstrap_dir)"
  if [ ! -f "${dir}/tfplan" ]; then
    log "Missing ${dir}/tfplan; run bootstrap plan first"
    exit 1
  fi
  tofu_show_plan "$dir" tfplan
  confirm_apply bootstrap
  tofu_apply_plan "$dir" tfplan
  log "Bootstrap apply complete"
}

bootstrap_plan_role_arn() {
  (cd "$(bootstrap_dir)" && tofu output -raw github_tofu_plan_role_arn)
}

bootstrap_apply_role_arn() {
  (cd "$(bootstrap_dir)" && tofu output -raw github_tofu_apply_role_arn)
}

github_setup_plan() {
  local local_pass staging_pass github_token plan_arn apply_arn dir
  local_pass="$(load_local_state_passphrase)"
  staging_pass="$(load_staging_state_passphrase)"
  github_token="$(load_github_admin_token)"
  plan_arn="$(bootstrap_plan_role_arn)"
  apply_arn="$(bootstrap_apply_role_arn)"
  export TF_VAR_local_state_encryption_passphrase="$local_pass"
  export TF_VAR_state_encryption_passphrase="$staging_pass"
  export TF_VAR_github_token="$github_token"
  export TF_VAR_tofu_plan_role_arn="$plan_arn"
  export TF_VAR_tofu_apply_role_arn="$apply_arn"
  dir="$(github_setup_dir)"
  tofu_init "$dir"
  tofu_plan_stack "$dir" tfplan
  log "GitHub setup plan written to ${dir}/tfplan"
}

github_setup_apply() {
  local local_pass staging_pass github_token plan_arn apply_arn dir
  local_pass="$(load_local_state_passphrase)"
  staging_pass="$(load_staging_state_passphrase)"
  github_token="$(load_github_admin_token)"
  plan_arn="$(bootstrap_plan_role_arn)"
  apply_arn="$(bootstrap_apply_role_arn)"
  export TF_VAR_local_state_encryption_passphrase="$local_pass"
  export TF_VAR_state_encryption_passphrase="$staging_pass"
  export TF_VAR_github_token="$github_token"
  export TF_VAR_tofu_plan_role_arn="$plan_arn"
  export TF_VAR_tofu_apply_role_arn="$apply_arn"
  dir="$(github_setup_dir)"
  if [ ! -f "${dir}/tfplan" ]; then
    log "Missing ${dir}/tfplan; run github-setup plan first"
    exit 1
  fi
  tofu_show_plan "$dir" tfplan
  confirm_apply github-setup
  tofu_apply_plan "$dir" tfplan
  log "GitHub setup apply complete"
}

maybe_push_baseline_tag() {
  [ "$PUSH_BASELINE_TAG" -eq 1 ] || return 0
  if git -C "$REPO_ROOT" ls-remote --exit-code --tags origin "refs/tags/${DEFAULT_BASELINE_TAG}" >/dev/null 2>&1; then
    log "${DEFAULT_BASELINE_TAG} already exists on origin"
    return 0
  fi
  git -C "$REPO_ROOT" tag -a "$DEFAULT_BASELINE_TAG" \
    -m "Pre remote-forward-fix baseline (main ${BASELINE_TAG_COMMIT})" "$BASELINE_TAG_COMMIT"
  git -C "$REPO_ROOT" push origin "$DEFAULT_BASELINE_TAG"
  log "Pushed ${DEFAULT_BASELINE_TAG}"
}

parse_global_args() {
  COMMAND=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --doppler-project) DOPPLER_PROJECT=$2; shift 2 ;;
      --doppler-config) DOPPLER_CONFIG=$2; shift 2 ;;
      --repo-root) REPO_ROOT=$2; shift 2 ;;
      --baseline-tag-commit) BASELINE_TAG_COMMIT=$2; shift 2 ;;
      --push-baseline-tag) PUSH_BASELINE_TAG=1; shift ;;
      --yes) AUTO_YES=1; shift ;;
      --dry-run) DRY_RUN=1; shift ;;
      -h|--help) usage ;;
      oidc|bootstrap|github-setup|all) COMMAND+=("$1"); shift ;;
      plan|apply|ensure) COMMAND+=("$1"); shift ;;
      *) log "unknown argument: $1"; usage ;;
    esac
  done
  if [ -z "$REPO_ROOT" ]; then
    REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
  fi
}

main() {
  parse_global_args "$@"
  if [ "${#COMMAND[@]}" -eq 0 ]; then
    usage
  fi

  if [ "$DRY_RUN" -eq 1 ]; then
    log "dry-run ok (doppler ${DOPPLER_PROJECT}/${DOPPLER_CONFIG}, command: ${COMMAND[*]})"
    exit 0
  fi

  if ! command -v doppler >/dev/null 2>&1; then
    log "doppler CLI is required (install or use doppler run -- $0 ...)"
    exit 1
  fi
  if ! command -v tofu >/dev/null 2>&1; then
    log "OpenTofu (tofu) is required"
    exit 1
  fi
  if ! command -v aws >/dev/null 2>&1; then
    log "aws CLI is required for bootstrap"
    exit 1
  fi
  if ! command -v jq >/dev/null 2>&1; then
    log "jq is required"
    exit 1
  fi

  case "${COMMAND[0]}" in
    oidc)
      [ "${COMMAND[1]:-}" = "ensure" ] || usage
      ensure_github_oidc_provider
      ;;
    bootstrap)
      case "${COMMAND[1]:-}" in
        plan) bootstrap_plan ;;
        apply) bootstrap_apply ;;
        *) usage ;;
      esac
      ;;
    github-setup)
      case "${COMMAND[1]:-}" in
        plan) github_setup_plan ;;
        apply) github_setup_apply ;;
        *) usage ;;
      esac
      ;;
    all)
      case "${COMMAND[1]:-}" in
        plan)
          bootstrap_plan
          github_setup_plan
          ;;
        apply)
          bootstrap_apply
          github_setup_apply
          maybe_push_baseline_tag
          ;;
        *) usage ;;
      esac
      ;;
    *) usage ;;
  esac
}

main "$@"
