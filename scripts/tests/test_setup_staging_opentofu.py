"""Tests for setup-staging-opentofu.sh with stubbed doppler, tofu, git, openssl."""
import json
import os
import shutil
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
REPO_ROOT = HERE.parent.parent
SCRIPT = HERE.parent / "setup-staging-opentofu.sh"
THUMBPRINTS_SRC = REPO_ROOT / "infra/opentofu/bootstrap/github-oidc-thumbprints"
SUBPROCESS_TIMEOUT = 30
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
        self.tofu_passphrase_log = Path(self.tmp) / "tofu-passphrases.log"
        self.aws_log = Path(self.tmp) / "aws.log"
        self.git_log = Path(self.tmp) / "git.log"
        self.aws_log.touch()
        self.git_log.touch()
        self.tofu_log.touch()
        self.tofu_passphrase_log.touch()
        if THUMBPRINTS_SRC.is_file():
            shutil.copy(THUMBPRINTS_SRC, self.bootstrap_dir / "github-oidc-thumbprints")
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
              if [ "${{STUB_DOPPLER_NOISY_SET:-}}" = "1" ]; then
                echo "Doppler CLI: updated $name successfully"
              fi
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
            if [ -n "${{TF_VAR_local_state_encryption_passphrase:-}}" ]; then
              echo "local_pass=${{TF_VAR_local_state_encryption_passphrase}}" >> "{self.tofu_passphrase_log}"
            fi
            if [ -n "${{TF_VAR_state_encryption_passphrase:-}}" ]; then
              echo "staging_pass=${{TF_VAR_state_encryption_passphrase}}" >> "{self.tofu_passphrase_log}"
            fi
            if [ -z "${{TF_VAR_local_state_encryption_passphrase:-}}" ]; then
              case "${{1:-}}" in init|plan|output|apply)
                echo "missing TF_VAR_local_state_encryption_passphrase" >&2
                exit 1 ;;
              esac
            fi
            BOOTSTRAP_MARKER="{self.bootstrap_dir}/.bootstrap_applied"
            IN_BOOTSTRAP=0
            if [ "$PWD" = "{self.bootstrap_dir}" ]; then IN_BOOTSTRAP=1; fi
            case "${{1:-}}" in
              init) exit 0 ;;
              plan)
                if [ "$IN_BOOTSTRAP" = "1" ]; then
                  create="${{TF_VAR_create_github_oidc_provider:-true}}"
                  if [ "$create" = "false" ] && [ "${{STUB_OIDC_EXISTS:-0}}" != "1" ]; then
                    echo "GitHub OIDC provider not found (create_github_oidc_provider=false)" >&2
                    exit 1
                  fi
                fi
                touch tfplan
                exit 0
                ;;
              show) exit 0 ;;
              apply)
                [ -f tfplan ] || exit 1
                if [ "$IN_BOOTSTRAP" = "1" ]; then touch "$BOOTSTRAP_MARKER"; fi
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
            timeout=SUBPROCESS_TIMEOUT,
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

    def test_bootstrap_plan_no_aws_cli_oidc(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root", str(self.h.repo_root), "bootstrap", "plan", STUB_OIDC_EXISTS="0"
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertNotIn("create-open-id-connect-provider", self.h.aws_log.read_text())

    def test_bootstrap_plan_detects_existing_oidc(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root", str(self.h.repo_root), "bootstrap", "plan", STUB_OIDC_EXISTS="1"
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("create_github_oidc_provider=false (existing GitHub OIDC provider found in AWS)", proc.stderr)

    def test_bootstrap_plan_detects_missing_oidc(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root", str(self.h.repo_root), "bootstrap", "plan", STUB_OIDC_EXISTS="0"
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("create_github_oidc_provider=true (no GitHub OIDC provider in AWS)", proc.stderr)

    def test_bootstrap_plan_override_create_false_fails_without_provider(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--create-github-oidc-provider",
            "false",
            "bootstrap",
            "plan",
            STUB_OIDC_EXISTS="0",
        )
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("create_github_oidc_provider=false (command-line override)", proc.stderr)

    def test_bootstrap_plan_on_fresh_account_default_create(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run(
            "--repo-root", str(self.h.repo_root), "bootstrap", "plan", STUB_OIDC_EXISTS="0"
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn(str(self.h.bootstrap_dir), self.h.tofu_log.read_text())

    def test_script_exports_passphrase_before_bootstrap_plan(self):
        self.h.doppler_file.write_text(json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": SECRET_LOCAL}))
        proc = self.h.run("--repo-root", str(self.h.repo_root), "bootstrap", "plan")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertNotIn("missing TF_VAR_local_state_encryption_passphrase", proc.stderr)

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
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "github-setup",
            "plan",
            STUB_DOPPLER_NOISY_SET="1",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        stored = self.h.secrets()["TOFU_STATE_ENCRYPTION_PASSPHRASE"]
        self.assertEqual(stored, GENERATED)
        self.h.assert_no_secrets_leaked(self, proc, GENERATED)
        pass_log = self.h.tofu_passphrase_log.read_text()
        self.assertIn(f"local_pass={SECRET_LOCAL}", pass_log)
        self.assertIn(f"staging_pass={GENERATED}", pass_log)
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
        self.assertIn("Bootstrap plan only. Next steps in order:", proc.stderr)
        self.assertNotIn(str(self.h.github_dir), self.h.tofu_log.read_text())

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
            STUB_OIDC_EXISTS="0",
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
            STUB_OIDC_EXISTS="0",
        )
        self.assertEqual(proc.returncode, 1)
        self.assertNotIn("create-open-id-connect-provider", self.h.aws_log.read_text())
        self.assertFalse((self.h.bootstrap_dir / ".bootstrap_applied").exists())

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
