"""Minimal stand-in for the Railway GraphQL API used by railway-deploy-image.sh tests.

Behaviour is driven by env vars:
  FAKE_STATUSES  comma-separated deployment statuses returned on successive polls
  FAKE_ERROR_ON  operation name (serviceInstanceUpdate / serviceInstanceDeployV2) that returns a GraphQL error
  FAKE_TOKEN     token the Project-Access-Token header must equal
  FAKE_LOG       file each request body is appended to (one JSON per line)
"""
import json
import os
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

statuses = os.environ.get("FAKE_STATUSES", "SUCCESS").split(",")
error_on = os.environ.get("FAKE_ERROR_ON", "")
token = os.environ.get("FAKE_TOKEN", "t")
log_path = os.environ["FAKE_LOG"]
polls = {"n": 0}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def _send(self, code, payload):
        data = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with open(log_path, "a") as f:
            f.write(json.dumps(body) + "\n")
        if self.headers.get("Project-Access-Token") != token:
            return self._send(401, {"errors": [{"message": "Not Authorized"}]})
        q = body["query"]
        for op in ("serviceInstanceUpdate", "serviceInstanceDeployV2"):
            if op in q and op == error_on:
                return self._send(200, {"errors": [{"message": f"{op} failed"}]})
        if "serviceInstanceUpdate" in q:
            return self._send(200, {"data": {"serviceInstanceUpdate": True}})
        if "serviceInstanceDeployV2" in q:
            return self._send(200, {"data": {"serviceInstanceDeployV2": "dep-123"}})
        if "deployment(" in q:
            i = min(polls["n"], len(statuses) - 1)
            polls["n"] += 1
            return self._send(200, {"data": {"deployment": {"status": statuses[i]}}})
        return self._send(400, {"errors": [{"message": "unknown query"}]})


if __name__ == "__main__":
    server = HTTPServer(("127.0.0.1", 0), Handler)
    print(server.server_address[1], flush=True)
    sys.stdout.close()
    server.serve_forever()
