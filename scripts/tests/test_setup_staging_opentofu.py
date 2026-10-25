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
        self._write_stubs()

    def _write_stubs(self) -> None:
        write_executable(
            self.bin_dir / "doppler",
            f"""\
            #!/usr/bin/env bash
            set -euo pipefail
            FILE="{self.doppler_file}"
            if [ "${{1:-}}" = "secrets" ] && [ "${{2:-}}" = "get" ]; then
              name="$3"
              val=$(jq -r --arg n "$name" '.[$n] // empty' "$FILE")
              if [ -z "$val" ]; then exit 1; fi
              printf '%s' "$val"
              exit 0
            fi
            if [ "${{1:-}}" = "secrets" ] && [ "${{2:-}}" = "set" ]; then
              name="$3"
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
            if [ "${{1:-}}" = "iam" ] && [ "${{2:-}}" = "list-open-id-connect-providers" ]; then
              if [ "${{STUB_OIDC_EXISTS:-0}}" = "1" ]; then
                echo '{{"OpenIDConnectProviderList":[{{"Arn":"arn:aws:iam::123:oidc-provider/token.actions.githubusercontent.com"}}]}}'
              else
                echo '{{"OpenIDConnectProviderList":[]}}'
              fi
              exit 0
            fi
            if [ "${{1:-}}" = "iam" ] && [ "${{2:-}}" = "create-open-id-connect-provider" ]; then
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
            cd "${{TOFU_STUB_CWD:-$(pwd)}}"
            case "${{1:-}}" in
              init) exit 0 ;;
              plan)
                touch tfplan
                exit 0
                ;;
              show) exit 0 ;;
              apply)
                if [ ! -f tfplan ]; then exit 1; fi
                exit 0
                ;;
              output)
                shift
                raw=0
                name=""
                while [ $# -gt 0 ]; do
                  case "$1" in
                    -raw) raw=1; shift ;;
                    *) name="$1"; shift ;;
                  esac
                done
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
            """\
            #!/usr/bin/env bash
            if [ "${1:-}" = "rand" ]; then echo "generated-passphrase-base64"; fi
            exit 0
            """,
        )

        write_executable(
            self.bin_dir / "git",
            f"""\
            #!/usr/bin/env bash
            echo "$*" >> "{self.git_log}"
            if echo "$*" | grep -q "ls-remote"; then
              exit 2
            fi
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
        env["STUB_DOPPLER_FILE"] = str(self.doppler_file)
        env.update(extra)
        return env

    def run(self, *args, input_text=None, **env_extra):
        return subprocess.run(
            ["bash", str(SCRIPT), *args],
            capture_output=True,
            text=True,
            env=self.env(**env_extra),
            input=input_text,
            timeout=30,
            cwd=str(self.repo_root),
        )

    def secrets(self):
        return json.loads(self.doppler_file.read_text())


class SetupStagingOpentofuStubTest(unittest.TestCase):
    def setUp(self):
        self.h = StubHarness()

    def test_dry_run_succeeds(self):
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--dry-run",
            "bootstrap",
            "plan",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr + proc.stdout)
        combined = proc.stdout + proc.stderr
        self.assertIn("dry-run ok", combined)

    def test_oidc_created_when_missing(self):
        proc = self.h.run("oidc", "ensure", STUB_OIDC_EXISTS="0")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        log = self.h.aws_log.read_text()
        self.assertIn("create-open-id-connect-provider", log)

    def test_oidc_skipped_when_exists(self):
        proc = self.h.run("oidc", "ensure", STUB_OIDC_EXISTS="1")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        log = self.h.aws_log.read_text()
        self.assertNotIn("create-open-id-connect-provider", log)

    def test_bootstrap_plan_stores_passphrase_in_doppler(self):
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "bootstrap",
            "plan",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        secrets = self.h.secrets()
        self.assertIn("LOCAL_STATE_ENCRYPTION_PASSPHRASE", secrets)
        self.assertIn("generated-passphrase-base64", secrets["LOCAL_STATE_ENCRYPTION_PASSPHRASE"])

    def test_bootstrap_plan_reuses_doppler_passphrase(self):
        self.h.doppler_file.write_text(
            json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": "existing-local-pass"})
        )
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "bootstrap",
            "plan",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(
            self.h.secrets()["LOCAL_STATE_ENCRYPTION_PASSPHRASE"],
            "existing-local-pass",
        )

    def test_apply_rejected_when_not_yes(self):
        (self.h.bootstrap_dir / "tfplan").touch()
        self.h.doppler_file.write_text(
            json.dumps({"LOCAL_STATE_ENCRYPTION_PASSPHRASE": "existing-local-pass"})
        )
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "bootstrap",
            "apply",
            input_text="no\n",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        self.assertIn("Apply cancelled", proc.stderr)

    def test_push_baseline_tag_flag(self):
        (self.h.bootstrap_dir / "tfplan").touch()
        self.h.doppler_file.write_text(
            json.dumps(
                {
                    "LOCAL_STATE_ENCRYPTION_PASSPHRASE": "local",
                    "TOFU_STATE_ENCRYPTION_PASSPHRASE": "staging",
                    "GITHUB_ADMIN_TOKEN": "ghp_test",
                }
            )
        )
        (self.h.github_dir / "tfplan").touch()
        proc = self.h.run(
            "--repo-root",
            str(self.h.repo_root),
            "--push-baseline-tag",
            "--yes",
            "all",
            "apply",
            STUB_OIDC_EXISTS="1",
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        git_log = self.h.git_log.read_text()
        self.assertIn("push", git_log)
        self.assertIn("hub-framework-staging-baseline", git_log)


if __name__ == "__main__":
    unittest.main()
