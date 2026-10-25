"""Argument and dry-run tests for setup-staging-opentofu.sh."""
import os
import subprocess
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPT = os.path.join(HERE, "..", "setup-staging-opentofu.sh")


class SetupStagingOpentofuTest(unittest.TestCase):
    def run_script(self, *args, extra_env=None):
        env = {k: v for k, v in os.environ.items() if not k.lower().endswith("_proxy")}
        env.setdefault("LOCAL_STATE_ENCRYPTION_PASSPHRASE", "test-local-pass")
        env.setdefault("GITHUB_ADMIN_TOKEN", "test-token")
        env.update(extra_env or {})
        return subprocess.run(
            ["bash", SCRIPT, *args],
            capture_output=True,
            text=True,
            env=env,
            timeout=30,
        )

    def test_dry_run_succeeds(self):
        proc = self.run_script("--dry-run")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("dry-run ok", proc.stdout)

    def test_unknown_argument_fails(self):
        proc = self.run_script("--not-a-flag")
        self.assertEqual(proc.returncode, 64)
        self.assertIn("unknown argument", proc.stderr)

    def test_missing_local_passphrase_fails(self):
        env = {
            "LOCAL_STATE_ENCRYPTION_PASSPHRASE": "",
            "GITHUB_ADMIN_TOKEN": "t",
        }
        proc = self.run_script("--dry-run", extra_env=env)
        self.assertNotEqual(proc.returncode, 0)

    def test_missing_github_token_fails(self):
        proc = self.run_script(
            "--dry-run",
            extra_env={"GITHUB_ADMIN_TOKEN": ""},
        )
        self.assertNotEqual(proc.returncode, 0)


if __name__ == "__main__":
    unittest.main()
