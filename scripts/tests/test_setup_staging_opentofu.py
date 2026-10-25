"""Tests for setup-staging-opentofu.sh with stubbed aws, doppler, tofu, git, openssl."""
import json
import os
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
SCRIPT = HERE.parent / "setup-staging-opentofu.sh"
SECRET_LOCAL = "existing-local-pass"
SECRET_STAGING = "existing-staging-pass"
GENERATED = "generated-passphrase-base64"


def write_executable(path: Path, content: str) -> None:
    path.write_text(textwrap.dedent(content).lstrip())
    path.chmod(path.stat().st_mode | stat.S_IEXEC)


class StubHarness:
    def __init__(self):
        self.tmp = tempfile.mkdtemp()
        self.bin_dir = Path(self.tmp) / "bin"
        self.bin_dir.mkdir()
        self.repo_root = Path(self.tmp) / "repo"
        self.bootstrap_dir = self.repo_root / "infra/opentofu/bootstrap"
        self.github_dir = self.repo_root / "infra/opentofu/github-setup"
        self.bootstrap_dir.mkdir(parents=True)
        self.github_dir.mkdir(parents=True)
        self.doppler_file = Path(self.tmp) / "doppler-secrets.json"
        self.doppler_file.write_text("{}")
        self.tofu_log = Path(self.tmp) / "tofu.log"
        self.aws_log = Path(self.tmp) / "aws.log"
        self.git_log = Path(self.tmp) / "git.log"
        self.aws_log.touch()
        self.git_log.touch()
        self.tofu_log.touch()
        self._write_stubs()

    def _write_stubs(self) -> None:
        doppler_file = self.doppler_file
        write_executable(
            self.bin_dir / "doppler",
            f"""\
            #!/usr/bin/env bash
            set -euo pipefail
            FILE="{doppler_file}"
            if [ "${{1:-}}" = "secrets" ] && [ "${{2:-}}" = "get" ]; then
              name="$3"
              if [ "${{STUB_DOPPLER_FAIL:-}}" = "network" ]; then
                echo "network timeout" >&2
                exit 2
              fi
              val=$(jq -r --arg n "$name" '.[$n] // empty' "$FILE")
              if [ -z "$val" ]; then
                echo "secret not found" >&2
                exit 1
              fi
              printf '%s' "$val"
              exit 0
            fi
            if [ "${{1:-}}" = "secrets" ] && [ "${{2:-}}" = "set" ]; then
              name="$3"
              shift 3
              while [ $# -gt 0 ]; do shift; done
              val=$(cat)
              tmp=$(mktemp)
              jq --arg k "$name" --arg v "$val" '.[$k]=$v' "$FILE" > "$tmp"
              mv "$tmp" "$FILE"
              exit 0
            fi
            echo "stub doppler unsupported: $*" >&2
            exit 1
            """,
        )

        write_executable(
            self.bin_dir / "aws",
            f"""\
            #!/usr/bin/env bash
            echo "$*" >> "{self.aws_log}"
            if echo "$*" | grep -q "list-open-id-connect-providers"; then
              if [ "${{STUB_OIDC_EXISTS:-0}}" = "1" ]; then
                echo '{{"OpenIDConnectProviderList":[{{"Arn":"arn:aws:iam::123:oidc-provider/token.actions.githubusercontent.com"}}]}}'
              else
                echo '{{"OpenIDConnectProviderList":[]}}'
              fi
              exit 0
            fi
            if echo "$*" | grep -q "create-open-id-connect-provider"; then
              exit 0
            fi
            exit 0
            """,
        )

        write_executable(
            self.bin_dir / "tofu",
            f"""\
            #!/usr/bin/env bash
            echo "$PWD $*" >> "{self.tofu_log}"
            if [ -z "${{TF_VAR_local_state_encryption_passphrase:-}}" ]; then
              if echo "$*" | grep -qE 'init|plan|output|apply'; then
                echo "missing TF_VAR_local_state_encryption_passphrase" >&2
                exit 1
              fi
            fi
            case "${{1:-}}" in
              init) exit 0 ;;
              plan) touch tfplan; exit 0 ;;
              show) exit 0 ;;
              apply)
                if [ ! -f tfplan ]; then exit 1; fi
                if echo "$PWD" | grep -q bootstrap; then
                  touch .bootstrap_applied
                fi
                exit 0
                ;;
              output)
                shift
                name=""
                while [ $# -gt 0 ]; do
                  case "$1" in
                    -raw) shift ;;
                    *) name="$1"; shift ;;
                  esac
                done
                if [ ! -f .bootstrap_applied ] && [ ! -f ../bootstrap/.bootstrap_applied ]; then
                  if [ ! -f .bootstrap_applied ]; then
                    if [ -f "$(dirname "$PWD")/bootstrap/.bootstrap_applied" ] || [ -f .bootstrap_applied ]; then
                      :
                    else
                      echo "bootstrap not applied" >&2
                      exit 1
                    fi
                  fi
                fi
                if [ -f .bootstrap_applied ] || [ -f "$(pwd)" ]/infra ]; then
                  :
                fi
                if [ -f .bootstrap_applied ]; then
                  if [ "$name" = "github_tofu_plan_role_arn" ]; then echo "arn:plan"; exit 0; fi
                  if [ "$name" = "github_tofu_apply_role_arn" ]; then echo "arn:apply"; exit 0; fi
                fi
                if [ -f .bootstrap_applied ]; then
                  :
                elif [ -f "$(dirname "$PWD")/.bootstrap_applied" ]; then
                  :
                fi
                if [ -f .bootstrap_applied ]; then
                  if [ "$name" = "github_tofu_plan_role_arn" ]; then echo "arn:plan"; exit 0; fi
                  if [ "$name" = "github_tofu_apply_role_arn" ]; then echo "arn:apply"; exit 0; fi
                fi
                if [ "$name" = "github_tofu_plan_role_arn" ] && [ -f .bootstrap_applied ]; then echo "arn:plan"; exit 0; fi
                if [ "$name" = "github_tofu_apply_role_arn" ] && [ -f .bootstrap_applied ]; then echo "arn:apply"; exit 0; fi
                if [ "$name" = "github_tofu_plan_role_arn" ] && [ -f "$PWD/.bootstrap_applied" ]; then echo "arn:plan"; exit 0; fi
                echo "bootstrap not applied" >&2
                exit 1
                ;;
            esac
            exit 0
            """,
        )

        # Simpler tofu stub rewrite inline below - the above is messy. Replace with clean version.
        write_executable(
            self.bin_dir / "tofu",
            f"""\
            #!/usr/bin/env bash
            echo "$PWD $*" >> "{self.tofu_log}"
            if [ -z "${{TF_VAR_local_state_encryption_passphrase:-}}" ]; then
              case "${{1:-}}" in init|plan|output|apply)
                echo "missing TF_VAR_local_state_encryption_passphrase" >&2
                exit 1 ;;
              esac
            fi
            BOOTSTRAP_MARKER="{self.bootstrap_dir}/.bootstrap_applied"
            case "${{1:-}}" in
              init) exit 0 ;;
              plan) touch tfplan; exit 0 ;;
              show) exit 0 ;;
              apply)
                [ -f tfplan ] || exit 1
                if [ "$PWD" = "{self.bootstrap_dir}" ]; then touch "$BOOTSTRAP_MARKER"; fi
                exit 0
                ;;
              output)
                shift
                name=""
                while [ $# -gt 0 ]; do
                  case "$1" in -raw) shift ;; *) name="$1"; shift ;; esac
                done
                [ -f "$BOOTSTRAP_MARKER" ] || {{ echo "bootstrap not applied" >&2; exit 1; }}
                if [ "$name" = "github_tofu_plan_role_arn" ]; then echo "arn:plan"; exit 0; fi
                if [ "$name" = "github_tofu_apply_role_arn" ]; then echo "arn:apply"; exit 0; fi
                exit 1
                ;;
            esac
            exit 0
            """,
        )

        write_executable(
            self.bin_dir / "openssl",
            f"""\
            #!/usr/bin/env bash
            if [ "${{1:-}}" = "rand" ]; then echo "{GENERATED}"; fi
            exit 0
            """,
        )

        write_executable(
            self.bin_dir / "git",
            f"""\
            #!/usr/bin/env bash
            echo "$*" >> "{self.git_log}"
            if echo "$*" | grep -q "ls-remote"; then exit 2; fi
            exit 0
            """,
        )

        write_executable(
            self.bin_dir / "jq",
            """\
            #!/usr/bin/env bash
            exec /usr/bin/jq "$@"
            """,
        )

    def env(self, **extra):
        env = {k: v for k, v in os.environ.items() if not k.lower().endswith("_proxy")}
        env["PATH"] = f"{self.bin_dir}:{env.get('PATH', '')}"
        env.update(extra)
        return env

    def run(self, *args, input_text=None, **env_extra):
        proc = subprocess.run(
            ["bash", str(SCRIPT), *args],
            capture_output=True,
            text=True,
            env=self.env(**env_extra),
            input=input_text,
            timeout=30,
            cwd=str(self.repo_root),
        )
        return proc

    def secrets(self):
        return json.loads(self.doppler_file.read_text())

    def combined_output(self, proc):
        return proc.stdout + proc.stderr

    def assert_no_secrets_leaked(self, testcase, proc, *secret_values):
        combined = self.combined_output(proc)
        for val in secret_values:
            if val and len(val) > 3:
                testcase.assertNotIn(val, combined)


class SetupStagingOpentofuStubTest(unittest.TestCase):
    def setUp(self):
        self.h = StubHarness()

    def base_secrets(self):
        self.h.doppler_file.write_text(
            json.dumps(
                {
                    "LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL,
                    "TOFU_STATE_ENCRYPTION_PASSPHRASE": SECRET_STAGING,
                    "GITHUB_ADMIN_TOKEN": "ghp_test_token_value",
                }
            )
        )

    def test_bootstrap_plan_no_aws_create(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root", str(self.h.repo_root), "bootstrap", "plan", STUB_OIDC_EXISTS="0"
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertNotIn("create-open-id-connect-provider", self.h.aws_log.read_text())

    def test_bootstrap_apply_creates_oidc_when_missing(self):
        (self.h.bootstrap_dir / "tfplan").touch()
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--yes",
            "bootstrap",
            "apply",
            STUB_OIDC_EXISTS="0",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("create-open-id-connect-provider", self.h.aws_log.read_text())

    def test_staging_passphrase_create_and_reuse(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        (self.h.bootstrap_dir / "tfplan").touch()
        self.h.run(
            "--repo-root", str(self.h.repo_root), "--yes", "bootstrap", "apply", STUB_OIDC_EXISTS="1"
        )
        self.h.doppler_file.write_text(
            json.dumps(
                {
                    "LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL,
                    "GITHUB_ADMIN_TOKEN": "ghp_test_token_value",
                }
            )
        )
        proc = self.h.run("--repo-root", str(self.h.repo_root), "github-setup", "plan")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(self.h.secrets()["TOFU_STATE_ENCRYPTION_PASSPHRASE"], GENERATED)
        self.h.doppler_file.write_text(
            json.dumps(
                {
                    "LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL,
                    "TOFU_STATE_ENCRYPTION_PASSPHRASE": SECRET_STAGING,
                    "GITHUB_ADMIN_TOKEN": "ghp_test_token_value",
                }
            )
        )
        proc2 = self.h.run("--repo-root", str(self.h.repo_root), "github-setup", "plan")
        self.assertEqual(proc2.returncode, 0)
        self.assertEqual(self.h.secrets()["TOFU_STATE_ENCRYPTION_PASSPHRASE"], SECRET_STAGING)

    def test_doppler_network_error_stops(self):
        proc = self.h.run(
            "--repo-root", str(self.h.repo_root), "bootstrap", "plan", STUB_DOPPLER_FAIL="network"
        )
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("Doppler error", proc.stderr)

    def test_all_plan_only_bootstrap(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run("--repo-root", str(self.h.repo_root), "all", "plan")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("Bootstrap plan only", proc.stderr)
        self.assertNotIn(f"{self.h.github_dir}/tfplan", self.h.tofu_log.read_text())

    def test_github_setup_without_bootstrap_fails(self):
        self.base_secrets()
        proc = self.h.run("--repo-root", str(self.h.repo_root), "github-setup", "plan")
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("Bootstrap outputs are missing", proc.stderr)

    def test_github_setup_alone_after_bootstrap_apply(self):
        self.base_secrets()
        (self.h.bootstrap_dir / "tfplan").touch()
        proc = self.h.run(
            "--repo-root", str(self.h.repo_root), "--yes", "bootstrap", "apply", STUB_OIDC_EXISTS="1"
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        proc2 = self.h.run("--repo-root", str(self.h.repo_root), "github-setup", "plan")
        self.assertEqual(proc2.returncode, 0, proc2.stderr)
        self.h.assert_no_secrets_leaked(self, proc2, SECRET_LOCAL, SECRET_STAGING, "ghp_test_token_value")

    def test_all_setup_full_flow(self):
        self.base_secrets()
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--yes",
            "all",
            "setup",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr + proc.stdout)
        self.assertTrue((self.h.github_dir / "tfplan").exists())
        self.assertIn("Revoke the fine-grained GitHub admin token", proc.stderr)

    def test_apply_rejected_stops(self):
        (self.h.bootstrap_dir / "tfplan").touch()
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "bootstrap",
            "apply",
            input_text="no\n",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 1)
        self.assertNotIn("create-open-id-connect-provider", self.h.aws_log.read_text())

    def test_push_baseline_flag_rejected_on_bootstrap_plan(self):
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--push-baseline-tag",
            "--dry-run",
            "bootstrap",
            "plan",
        )
        self.assertEqual(proc.returncode, 1)
        self.assertIn("--push-baseline-tag", proc.stderr)

    def test_baseline_flag_off_no_git_push(self):
        self.base_secrets()
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--yes",
            "all",
            "setup",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertNotIn("push", self.h.git_log.read_text())

    def test_tofu_stub_fails_without_passphrase(self):
        env = {k: v for k, v in os.environ.items() if k != "TF_VAR_local_state_encryption_passphrase"}
        env["PATH"] = str(self.h.bin_dir)
        proc = subprocess.run(
            ["/bin/bash", str(self.h.bin_dir / "tofu"), "plan"],
            capture_output=True,
            text=True,
            env=env,
            cwd=str(self.h.bootstrap_dir),
        )
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("missing TF_VAR_local_state_encryption_passphrase", proc.stderr)

    def test_push_baseline_tag_on_all_setup(self):
        self.base_secrets()
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--yes",
            "--push-baseline-tag",
            "all",
            "setup",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("push", self.h.git_log.read_text())


if __name__ == "__main__":
    unittest.main()
