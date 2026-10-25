#!/usr/bin/env bash
# End-to-end smoke: hub-app forwards a POST to python-handler exampleTaskPython.
set -euo pipefail

HUB_URL="${HUB_URL:-http://localhost:8080}"
payload='{"name":"Smoke Cake","ingredients":["flour"],"steps":["mix"],"comments":[]}'

response="$(curl -fsS -X POST "${HUB_URL}/smokeApp/forward" \
  -H 'Content-Type: application/json' \
  -d "${payload}")"

echo "${response}" | tee /dev/stderr | grep -q 'a little extra love'
