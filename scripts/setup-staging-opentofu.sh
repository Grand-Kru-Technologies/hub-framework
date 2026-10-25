#!/usr/bin/env bash
# Library-style OpenTofu staging setup (bootstrap + github-setup).
# Prefer: doppler run --project hub-framework --config staging -- ./scripts/setup-staging-opentofu.sh ...
#
# First-time flow (four steps):
#   bootstrap plan -> bootstrap apply -> github-setup plan -> github-setup apply
# Or one shot:  all setup
#
set -euo pipefail

DEFAULT_DOPPLER_PROJECT="hub-framework"
DEFAULT_DOPPLER_CONFIG="staging"
DEFAULT_BASELINE_COMMIT="1583326b"
DEFAULT_BASELINE_TAG="hub-framework-staging-baseline"
GITHUB_OIDC_URL="https://token.actions.githubusercontent.com"
CREATE_GITHUB_OIDC_PROVIDER_OVERRIDE=""

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
  oidc ensure                   Manual fallback: create GitHub OIDC via AWS CLI (confirmed)
  bootstrap plan                Plan bootstrap (includes OIDC provider when create_github_oidc_provider=true)
  bootstrap apply               Apply bootstrap plan
  github-setup plan             Plan github-setup (bootstrap must be applied)
  github-setup apply            Apply github-setup plan
  all plan                      Bootstrap plan only (then run bootstrap apply)
  all apply                     Bootstrap apply then github-setup apply (both tfplans required)
  all setup                     Full flow: bootstrap plan/apply then github-setup plan/apply

Options:
  --doppler-project NAME
  --doppler-config NAME
  --repo-root PATH
  --baseline-tag-commit SHA
  --push-baseline-tag           Only valid with all apply or all setup
  --create-github-oidc-provider true|false
                                Override bootstrap OIDC mode (default: detect via AWS)
  --yes                         Skip apply confirmation (unsafe)
  --dry-run
  -h, --help
EOF
  exit 64
}

log() { printf '%s\n' "$*" >&2; }

doppler_cli() {
  doppler "$@" --project "$DOPPLER_PROJECT" --config "$DOPPLER_CONFIG"
}

doppler_secret_not_found() {
  local err_file=$1
  grep -qiE 'not found|could not find|does not exist|404|not exist' "$err_file"
}

# Returns 0 and prints value; 1 if secret missing; 2 on other Doppler errors (message on stderr).
doppler_secret_get() {
  local name=$1
  local err out
  err="$(mktemp)"
  trap 'rm -f "$err"' RETURN
  if out="$(doppler_cli secrets get "$name" --plain 2>"$err")"; then
    printf '%s' "$out"
    return 0
  fi
  if doppler_secret_not_found "$err"; then
    return 1
  fi
  cat "$err" >&2
  log "Doppler error while reading ${name}; not generating a new value"
  return 2
}

doppler_secret_set() {
  local name=$1 value=$2
  if ! printf '%s' "$value" | doppler_cli secrets set "$name" --silent >/dev/null; then
    log "Doppler error while writing ${name}"
    exit 1
  fi
}

ensure_doppler_secret() {
  local name=$1 generator=$2
  local value rc=0
  value="$(doppler_secret_get "$name")" || rc=$?
  if [ "$rc" -eq 0 ]; then
    printf '%s' "$value"
    return 0
  fi
  if [ "$rc" -eq 2 ]; then
    exit 1
  fi
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
  local value rc=0
  value="$(doppler_secret_get "$SECRET_GITHUB_ADMIN")" || rc=$?
  if [ "$rc" -eq 0 ]; then
    printf '%s' "$value"
    return 0
  fi
  if [ "$rc" -eq 1 ]; then
    log "Doppler secret ${SECRET_GITHUB_ADMIN} is required for github-setup"
    exit 1
  fi
  exit 1
}

export_local_state_passphrase() {
  TF_VAR_local_state_encryption_passphrase="$(load_local_state_passphrase)"
  export TF_VAR_local_state_encryption_passphrase
}

github_oidc_thumbprints_file() {
  printf '%s/infra/opentofu/bootstrap/github-oidc-thumbprints' "$REPO_ROOT"
}

read_github_oidc_thumbprints() {
  local file thumbprints
  file="$(github_oidc_thumbprints_file)"
  if [ ! -f "$file" ]; then
    log "Missing ${file}"
    exit 1
  fi
  thumbprints="$(tr -d '[:space:]' <"$file")"
  printf '%s' "$thumbprints" | tr ',' ' '
}

github_oidc_provider_exists_in_aws() {
  aws iam list-open-id-connect-providers --output json \
    | jq -e '.OpenIDConnectProviderList[] | select(.Arn | contains("token.actions.githubusercontent.com"))' >/dev/null
}

bootstrap_state_manages_github_oidc() {
  export_local_state_passphrase
  (cd "$(bootstrap_dir)" && tofu state list -no-color 2>/dev/null \
    | grep -qE '^aws_iam_openid_connect_provider\.github(\[0\])?$')
}

resolve_create_github_oidc_provider_for_plan() {
  export_local_state_passphrase
  if [ -n "$CREATE_GITHUB_OIDC_PROVIDER_OVERRIDE" ]; then
    TF_VAR_create_github_oidc_provider="$CREATE_GITHUB_OIDC_PROVIDER_OVERRIDE"
    log "create_github_oidc_provider=${TF_VAR_create_github_oidc_provider} (command-line override)"
  elif bootstrap_state_manages_github_oidc; then
    TF_VAR_create_github_oidc_provider=true
    log "create_github_oidc_provider=true (bootstrap state already manages GitHub OIDC provider)"
  elif github_oidc_provider_exists_in_aws; then
    TF_VAR_create_github_oidc_provider=false
    log "create_github_oidc_provider=false (existing GitHub OIDC provider found in AWS)"
  else
    TF_VAR_create_github_oidc_provider=true
    log "create_github_oidc_provider=true (no GitHub OIDC provider in AWS)"
  fi
  export TF_VAR_create_github_oidc_provider
}

ensure_github_oidc_provider() {
  local -a oidc_thumbs
  if github_oidc_provider_exists_in_aws; then
    log "GitHub OIDC provider already exists"
    return 0
  fi
  log "Creating GitHub OIDC provider"
  read -ra oidc_thumbs <<< "$(read_github_oidc_thumbprints)"
  aws iam create-open-id-connect-provider \
    --url "$GITHUB_OIDC_URL" \
    --client-id-list sts.amazonaws.com \
    --thumbprint-list "${oidc_thumbs[@]}" >/dev/null
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
  (cd "$dir" && tofu plan -input=false -no-color -out="$out")
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
  resolve_create_github_oidc_provider_for_plan
  local dir
  dir="$(bootstrap_dir)"
  tofu_init "$dir"
  tofu_plan_stack "$dir" tfplan
  log "Bootstrap plan written to ${dir}/tfplan"
}

bootstrap_apply() {
  export_local_state_passphrase
  local dir
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

bootstrap_has_outputs() {
  export_local_state_passphrase
  (cd "$(bootstrap_dir)" && tofu output -raw github_tofu_plan_role_arn >/dev/null 2>&1)
}

bootstrap_plan_role_arn() {
  export_local_state_passphrase
  (cd "$(bootstrap_dir)" && tofu output -raw github_tofu_plan_role_arn)
}

bootstrap_apply_role_arn() {
  export_local_state_passphrase
  (cd "$(bootstrap_dir)" && tofu output -raw github_tofu_apply_role_arn)
}

prepare_github_setup_tfvars() {
  export_local_state_passphrase
  local staging_pass github_token
  staging_pass="$(load_staging_state_passphrase)"
  github_token="$(load_github_admin_token)"
  export TF_VAR_state_encryption_passphrase="$staging_pass"
  export TF_VAR_github_token="$github_token"
  if ! bootstrap_has_outputs; then
    log "Bootstrap outputs are missing. Run bootstrap plan, then bootstrap apply, before github-setup."
    exit 1
  fi
  local plan_role apply_role
  plan_role="$(bootstrap_plan_role_arn)"
  apply_role="$(bootstrap_apply_role_arn)"
  export TF_VAR_tofu_plan_role_arn="$plan_role"
  export TF_VAR_tofu_apply_role_arn="$apply_role"
}

github_setup_plan() {
  prepare_github_setup_tfvars
  local dir
  dir="$(github_setup_dir)"
  tofu_init "$dir"
  tofu_plan_stack "$dir" tfplan
  log "GitHub setup plan written to ${dir}/tfplan"
}

github_setup_apply() {
  prepare_github_setup_tfvars
  local dir
  dir="$(github_setup_dir)"
  if [ ! -f "${dir}/tfplan" ]; then
    log "Missing ${dir}/tfplan; run github-setup plan first"
    exit 1
  fi
  tofu_show_plan "$dir" tfplan
  confirm_apply github-setup
  tofu_apply_plan "$dir" tfplan
  log "GitHub setup apply complete"
  print_post_setup_reminder
}

print_post_setup_reminder() {
  log "Setup complete."
  log "Revoke the fine-grained GitHub admin token: remove ${SECRET_GITHUB_ADMIN} from Doppler and revoke the token in GitHub settings."
  log "Use a repo-scoped token with minimal permissions (administration, environments, secrets, variables) and short expiry when re-running setup."
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

all_plan() {
  bootstrap_plan
  log "Bootstrap plan only. Next steps in order:"
  log "  1) $(basename "$0") bootstrap apply"
  log "  2) $(basename "$0") github-setup plan"
  log "  3) $(basename "$0") github-setup apply"
}

all_apply() {
  bootstrap_apply
  github_setup_apply
  maybe_push_baseline_tag
}

all_setup() {
  bootstrap_plan
  bootstrap_apply
  github_setup_plan
  github_setup_apply
  maybe_push_baseline_tag
}

validate_push_baseline_flag() {
  [ "$PUSH_BASELINE_TAG" -eq 0 ] && return 0
  case "${COMMAND[0]}-${COMMAND[1]:-}" in
    all-apply|all-setup) return 0 ;;
    *)
      log "error: --push-baseline-tag is only valid with 'all apply' or 'all setup'"
      exit 1
      ;;
  esac
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
      --create-github-oidc-provider)
        CREATE_GITHUB_OIDC_PROVIDER_OVERRIDE=$2
        shift 2
        ;;
      --yes) AUTO_YES=1; shift ;;
      --dry-run) DRY_RUN=1; shift ;;
      -h|--help) usage ;;
      oidc|bootstrap|github-setup|all) COMMAND+=("$1"); shift ;;
      plan|apply|ensure|setup) COMMAND+=("$1"); shift ;;
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
  validate_push_baseline_flag

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
  if ! command -v jq >/dev/null 2>&1; then
    log "jq is required"
    exit 1
  fi

  case "${COMMAND[0]}-${COMMAND[1]:-}" in
    bootstrap-plan|all-plan|all-setup)
      if ! command -v aws >/dev/null 2>&1; then
        log "aws CLI is required for bootstrap plan (OIDC provider detection)"
        exit 1
      fi
      ;;
  esac

  case "${COMMAND[0]}" in
    oidc)
      [ "${COMMAND[1]:-}" = "ensure" ] || usage
      if ! command -v aws >/dev/null 2>&1; then
        log "aws CLI is required for oidc ensure"
        exit 1
      fi
      log "Fallback only: prefer bootstrap OpenTofu (create_github_oidc_provider=true) so OIDC appears in the plan you approve."
      confirm_apply "GitHub OIDC provider (AWS CLI fallback)"
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
        plan) all_plan ;;
        apply) all_apply ;;
        setup) all_setup ;;
        *) usage ;;
      esac
      ;;
    *) usage ;;
  esac
}

main "$@"
