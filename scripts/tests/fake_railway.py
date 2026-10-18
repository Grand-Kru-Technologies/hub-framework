"""Minimal stand-in for the Railway GraphQL API used by railway-wait-deployment.sh tests.

Behaviour is driven by env vars:
  FAKE_DEPLOYMENTS  JSON list of deployment nodes ({id,status,createdAt}) or null,
                    returned on successive polls (last one repeats)
  FAKE_ERROR        if set, every request returns this GraphQL error message
  FAKE_TOKEN        value the Project-Access-Token header must equal
  FAKE_LOG          file each request body is appended to (one JSON per line)
"""
import json
import os
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

deployments = json.loads(os.environ.get("FAKE_DEPLOYMENTS", "[]"))
error = os.environ.get("FAKE_ERROR", "")
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
        if error:
            return self._send(200, {"errors": [{"message": error}]})
        if "deployments(" not in body["query"]:
            return self._send(400, {"errors": [{"message": "unknown query"}]})
        node = deployments[min(polls["n"], len(deployments) - 1)] if deployments else None
        polls["n"] += 1
        edges = [{"node": node}] if node else []
        return self._send(200, {"data": {"deployments": {"edges": edges}}})


if __name__ == "__main__":
    server = HTTPServer(("127.0.0.1", 0), Handler)
    print(server.server_address[1], flush=True)
    sys.stdout.close()
    server.serve_forever()
