"""Integration tests: run railway-wait-deployment.sh against a fake Railway API."""
import json
import os
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPT = os.path.join(HERE, "..", "railway-wait-deployment.sh")
FAKE = os.path.join(HERE, "fake_railway.py")


def dep(status, created="2026-10-18T12:00:00.000Z", id_="dep-1"):
    return {"id": id_, "status": status, "createdAt": created}


class RailwayWaitDeploymentTest(unittest.TestCase):
    def run_script(self, deployments, args=("svc-1",), error="", token="t", extra_env=None):
        log = tempfile.NamedTemporaryFile(delete=False, suffix=".jsonl")
        log.close()
        self.addCleanup(os.unlink, log.name)
        env_fake = dict(os.environ, FAKE_DEPLOYMENTS=json.dumps(deployments), FAKE_ERROR=error,
                        FAKE_TOKEN="t", FAKE_LOG=log.name)
        fake = subprocess.Popen([sys.executable, "-I", FAKE], stdout=subprocess.PIPE,
                                env=env_fake, text=True)
        port = fake.stdout.readline().strip()
        fake.stdout.close()
        self.addCleanup(fake.wait)
        self.addCleanup(fake.kill)
        env = {k: v for k, v in os.environ.items() if not k.lower().endswith("_proxy")}
        env.update(RAILWAY_TOKEN=token, RAILWAY_ENVIRONMENT_ID="env-1",
                   RAILWAY_API_URL=f"http://127.0.0.1:{port}/graphql/v2",
                   RAILWAY_POLL_INTERVAL="0", RAILWAY_DEPLOY_TIMEOUT="5")
        env.update(extra_env or {})
        proc = subprocess.run(["bash", SCRIPT, *args], capture_output=True, text=True,
                              env=env, timeout=30)
        with open(log.name) as f:
            calls = [json.loads(line) for line in f if line.strip()]
        return proc, calls

    def test_waits_through_build_until_success(self):
        proc, calls = self.run_script([dep("BUILDING"), dep("DEPLOYING"), dep("SUCCESS")])
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(len(calls), 3)
        self.assertEqual(calls[0]["variables"], {"serviceId": "svc-1", "environmentId": "env-1"})
        self.assertIn("dep-1: SUCCESS", proc.stdout)

    def test_failed_exits_1(self):
        proc, _ = self.run_script([dep("BUILDING"), dep("FAILED")])
        self.assertEqual(proc.returncode, 1)

    def test_crashed_exits_1(self):
        proc, _ = self.run_script([dep("CRASHED")])
        self.assertEqual(proc.returncode, 1)

    def test_ignores_deployments_older_than_not_before(self):
        old = dep("SUCCESS", created="2026-10-18T11:00:00.000Z", id_="old")
        new = dep("SUCCESS", created="2026-10-18T12:30:00.000Z", id_="new")
        proc, calls = self.run_script([old, old, new], args=("svc-1", "2026-10-18T12:00:00Z"))
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(len(calls), 3)
        self.assertIn("deployment new: SUCCESS", proc.stdout)
        self.assertNotIn("deployment old", proc.stdout)

    def test_no_deployment_yet_then_success(self):
        proc, _ = self.run_script([None, dep("SUCCESS")])
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("waiting for a new deployment", proc.stdout)

    def test_graphql_error_fails(self):
        proc, _ = self.run_script([dep("SUCCESS")], error="Project not found")
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("Project not found", proc.stderr)

    def test_bad_token_fails(self):
        proc, _ = self.run_script([dep("SUCCESS")], token="wrong")
        self.assertNotEqual(proc.returncode, 0)

    def test_timeout(self):
        proc, _ = self.run_script([dep("BUILDING")],
                                  extra_env={"RAILWAY_DEPLOY_TIMEOUT": "1", "RAILWAY_POLL_INTERVAL": "1"})
        self.assertEqual(proc.returncode, 1)
        self.assertIn("timed out", proc.stderr)

    def test_usage_error(self):
        proc = subprocess.run(["bash", SCRIPT], capture_output=True, text=True)
        self.assertEqual(proc.returncode, 64)

    def test_missing_token(self):
        env = {k: v for k, v in os.environ.items() if k != "RAILWAY_TOKEN"}
        env["RAILWAY_ENVIRONMENT_ID"] = "env-1"
        proc = subprocess.run(["bash", SCRIPT, "s"], capture_output=True, text=True, env=env)
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("RAILWAY_TOKEN is required", proc.stderr)


if __name__ == "__main__":
    unittest.main()
