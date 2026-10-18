#!/usr/bin/env bash
# GraphQL variables ($serviceId etc.) are intentionally literal in single quotes.
# shellcheck disable=SC2016
#
# Wait for the newest Railway deployment of a service to reach SUCCESS.
# Railway marks a deployment SUCCESS only after its healthcheck (/health) passes,
# so exit 0 means the service is up and healthy.
#
# Usage: railway-wait-deployment.sh <service-id> [<not-before-iso8601>]
#   <not-before-iso8601>  ignore deployments created before this time (default: none)
#
# Env:
#   RAILWAY_TOKEN           Railway project token (required)
#   RAILWAY_ENVIRONMENT_ID  environment (required)
#   RAILWAY_API_URL         GraphQL endpoint (default https://backboard.railway.com/graphql/v2)
#   RAILWAY_DEPLOY_TIMEOUT  seconds to wait (default 900)
#   RAILWAY_POLL_INTERVAL   seconds between polls (default 10)
set -euo pipefail

usage() { echo "usage: $0 <service-id> [<not-before-iso8601>]" >&2; exit 64; }
[ $# -ge 1 ] && [ $# -le 2 ] || usage
SERVICE_ID=$1
NOT_BEFORE=${2:-}
: "${RAILWAY_TOKEN:?RAILWAY_TOKEN is required}"
: "${RAILWAY_ENVIRONMENT_ID:?RAILWAY_ENVIRONMENT_ID is required}"
API=${RAILWAY_API_URL:-https://backboard.railway.com/graphql/v2}
TIMEOUT=${RAILWAY_DEPLOY_TIMEOUT:-900}
INTERVAL=${RAILWAY_POLL_INTERVAL:-10}

QUERY='query($serviceId:String!,$environmentId:String!){deployments(first:1,input:{serviceId:$serviceId,environmentId:$environmentId}){edges{node{id status createdAt}}}}'
VARS=$(jq -cn --arg s "$SERVICE_ID" --arg e "$RAILWAY_ENVIRONMENT_ID" '{serviceId:$s, environmentId:$e}')
BODY=$(jq -cn --arg q "$QUERY" --argjson v "$VARS" '{query:$q, variables:$v}')

latest() {
  local resp
  resp=$(curl -fsS -X POST "$API" \
    -H "Content-Type: application/json" \
    -H "Project-Access-Token: $RAILWAY_TOKEN" \
    --data "$BODY")
  if [ "$(jq '(.errors // []) | length' <<<"$resp")" != "0" ]; then
    echo "railway api error: $(jq -c '.errors' <<<"$resp")" >&2
    return 1
  fi
  jq -r '.data.deployments.edges[0].node // {} | "\(.id // "") \(.status // "") \(.createdAt // "")"' <<<"$resp"
}

deadline=$(( $(date +%s) + TIMEOUT ))
while :; do
  read -r id status created <<<"$(latest)"
  if [ -z "$id" ] || { [ -n "$NOT_BEFORE" ] && [[ "$created" < "$NOT_BEFORE" ]]; }; then
    echo "service $SERVICE_ID: waiting for a new deployment"
  else
    echo "service $SERVICE_ID deployment $id: $status"
    case "$status" in
      SUCCESS) exit 0 ;;
      FAILED|CRASHED|REMOVED|SKIPPED) exit 1 ;;
    esac
  fi
  [ "$(date +%s)" -lt "$deadline" ] || { echo "timed out after ${TIMEOUT}s" >&2; exit 1; }
  sleep "$INTERVAL"
done
