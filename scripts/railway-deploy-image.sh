#!/usr/bin/env bash
# GraphQL variables ($serviceId etc.) are intentionally literal in single quotes.
# shellcheck disable=SC2016
# Point a Railway service at a container image, deploy it, and wait until the
# deployment reports SUCCESS (Railway only marks SUCCESS after its healthcheck
# passes).
#
# Usage: railway-deploy-image.sh <service-id> <image>
#
# Env:
#   RAILWAY_TOKEN           Railway project token (required)
#   RAILWAY_ENVIRONMENT_ID  environment to deploy into (required)
#   RAILWAY_API_URL         GraphQL endpoint (default https://backboard.railway.com/graphql/v2)
#   RAILWAY_DEPLOY_TIMEOUT  seconds to wait for SUCCESS (default 600)
#   RAILWAY_POLL_INTERVAL   seconds between status polls (default 10)
set -euo pipefail

usage() { echo "usage: $0 <service-id> <image>" >&2; exit 64; }
[ $# -eq 2 ] || usage
SERVICE_ID=$1
IMAGE=$2
: "${RAILWAY_TOKEN:?RAILWAY_TOKEN is required}"
: "${RAILWAY_ENVIRONMENT_ID:?RAILWAY_ENVIRONMENT_ID is required}"
API=${RAILWAY_API_URL:-https://backboard.railway.com/graphql/v2}
TIMEOUT=${RAILWAY_DEPLOY_TIMEOUT:-600}
INTERVAL=${RAILWAY_POLL_INTERVAL:-10}

gql() {
  local query=$1 variables=$2 body resp
  body=$(jq -cn --arg q "$query" --argjson v "$variables" '{query:$q, variables:$v}')
  resp=$(curl -fsS -X POST "$API" \
    -H "Content-Type: application/json" \
    -H "Project-Access-Token: $RAILWAY_TOKEN" \
    --data "$body")
  if [ "$(jq '(.errors // []) | length' <<<"$resp")" != "0" ]; then
    echo "railway api error: $(jq -c '.errors' <<<"$resp")" >&2
    return 1
  fi
  jq -c '.data' <<<"$resp"
}

vars=$(jq -cn --arg s "$SERVICE_ID" --arg e "$RAILWAY_ENVIRONMENT_ID" --arg i "$IMAGE" \
  '{serviceId:$s, environmentId:$e, input:{source:{image:$i}}}')
gql 'mutation($serviceId:String!,$environmentId:String,$input:ServiceInstanceUpdateInput!){serviceInstanceUpdate(serviceId:$serviceId,environmentId:$environmentId,input:$input)}' "$vars" >/dev/null
echo "service $SERVICE_ID -> $IMAGE"

vars=$(jq -cn --arg s "$SERVICE_ID" --arg e "$RAILWAY_ENVIRONMENT_ID" '{serviceId:$s, environmentId:$e}')
deployment_id=$(gql 'mutation($serviceId:String!,$environmentId:String!){serviceInstanceDeployV2(serviceId:$serviceId,environmentId:$environmentId)}' "$vars" | jq -r '.serviceInstanceDeployV2')
[ -n "$deployment_id" ] && [ "$deployment_id" != "null" ] || { echo "no deployment id returned" >&2; exit 1; }
echo "deployment $deployment_id started"

deadline=$(( $(date +%s) + TIMEOUT ))
while :; do
  status=$(gql 'query($id:String!){deployment(id:$id){status}}' "$(jq -cn --arg id "$deployment_id" '{id:$id}')" | jq -r '.deployment.status')
  echo "deployment $deployment_id: $status"
  case "$status" in
    SUCCESS) exit 0 ;;
    FAILED|CRASHED|REMOVED|SKIPPED) exit 1 ;;
  esac
  [ "$(date +%s)" -lt "$deadline" ] || { echo "timed out after ${TIMEOUT}s" >&2; exit 1; }
  sleep "$INTERVAL"
done
