"""Integration tests: run railway-deploy-image.sh against a fake Railway API."""
import json
import os
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPT = os.path.join(HERE, "..", "railway-deploy-image.sh")
FAKE = os.path.join(HERE, "fake_railway.py")


class RailwayDeployImageTest(unittest.TestCase):
    def run_script(self, args=("svc-1", "ghcr.io/o/r:sha-1"), statuses="SUCCESS",
                   error_on="", token="t", server_token="t", extra_env=None):
        log = tempfile.NamedTemporaryFile(delete=False, suffix=".jsonl")
        log.close()
        env_fake = dict(os.environ, FAKE_STATUSES=statuses, FAKE_ERROR_ON=error_on,
                        FAKE_TOKEN=server_token, FAKE_LOG=log.name)
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
        os.unlink(log.name)
        return proc, calls

    def test_success_sets_image_then_deploys_then_polls(self):
        proc, calls = self.run_script(statuses="BUILDING,DEPLOYING,SUCCESS")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("serviceInstanceUpdate", calls[0]["query"])
        self.assertEqual(calls[0]["variables"],
                         {"serviceId": "svc-1", "environmentId": "env-1",
                          "input": {"source": {"image": "ghcr.io/o/r:sha-1"}}})
        self.assertIn("serviceInstanceDeployV2", calls[1]["query"])
        self.assertEqual(sum("deployment(" in c["query"] for c in calls), 3)
        self.assertIn("dep-123: SUCCESS", proc.stdout)

    def test_failed_deployment_exits_nonzero(self):
        proc, _ = self.run_script(statuses="BUILDING,FAILED")
        self.assertEqual(proc.returncode, 1)

    def test_crashed_deployment_exits_nonzero(self):
        proc, _ = self.run_script(statuses="CRASHED")
        self.assertEqual(proc.returncode, 1)

    def test_graphql_error_on_update_stops_before_deploy(self):
        proc, calls = self.run_script(error_on="serviceInstanceUpdate")
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("serviceInstanceUpdate failed", proc.stderr)
        self.assertEqual(len(calls), 1)

    def test_bad_token_fails(self):
        proc, _ = self.run_script(token="wrong")
        self.assertNotEqual(proc.returncode, 0)

    def test_timeout(self):
        proc, _ = self.run_script(statuses="BUILDING", extra_env={"RAILWAY_DEPLOY_TIMEOUT": "1",
                                                                  "RAILWAY_POLL_INTERVAL": "1"})
        self.assertEqual(proc.returncode, 1)
        self.assertIn("timed out", proc.stderr)

    def test_missing_args_is_usage_error(self):
        proc = subprocess.run(["bash", SCRIPT], capture_output=True, text=True)
        self.assertEqual(proc.returncode, 64)

    def test_missing_token_fails_without_network(self):
        env = {k: v for k, v in os.environ.items() if k != "RAILWAY_TOKEN"}
        env["RAILWAY_ENVIRONMENT_ID"] = "env-1"
        proc = subprocess.run(["bash", SCRIPT, "s", "i"], capture_output=True, text=True, env=env)
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("RAILWAY_TOKEN is required", proc.stderr)


if __name__ == "__main__":
    unittest.main()
