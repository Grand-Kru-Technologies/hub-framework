# hub-framework runbook

Covers the example stack in this repo: **hub-app** (`example/recipe-app`, Go) and **python-handler** (`integrations/python`, Flask), as wired in `docker-compose.yml`.

| | Local | Staging |
|---|---|---|
| hub-app | http://localhost:8080 | https://staging.hub.grandkru.com |
| python-handler | http://localhost:5000 | private only: `python-handler.railway.internal:5000` |
| Hosting | Docker Compose | Railway project `qg-hub-staging`, environment `staging` |

## 1. Run locally

Requirements: Docker with Compose v2.

```sh
docker compose up --build --wait      # builds both images, returns when both healthchecks pass
docker compose ps                     # both services "healthy"
docker compose logs -f                # follow logs
docker compose down -v                # stop and clean up
```

Without Docker:

```sh
# python-handler
python3 -m venv .venv && .venv/bin/pip install -r integrations/python/requirements.txt
PROJECT_ROOT=$PWD YAML_FILE_PATH=$PWD/example/recipe-app/python/handlers.yaml \
  .venv/bin/python integrations/python/start.py            # :5000

# hub-app (second shell)
cd example/recipe-app && PYTHON_HANDLER_URL=http://localhost:5000 go run .   # :8080
```

Test suites:

```sh
go build ./... && go vet ./... && go test ./... -count=1               # Go unit + integration
(cd integrations/python && python -m unittest discover -s tests -v)    # Python unit
python -m unittest discover -s scripts/tests -v                        # deploy script integration
shellcheck scripts/*.sh
```

## 2. How CI proves it

`.github/workflows/ci.yml` runs on every pull request and on every push to `main`:

| Job | Layer | Proves |
|---|---|---|
| `go` | unit + integration | `go build`, `go vet`, `go test ./...` across the framework and the recipe app |
| `python` | unit | handler loader, request handler and start-up tests |
| `deploy script` | integration | `shellcheck` plus `scripts/railway-wait-deployment.sh` against a fake Railway API (success, failure, crash, stale deployment, timeout, auth) |
| `docker compose smoke test` | end to end | both images build, `docker compose up --wait` turns both healthchecks green, `GET :8080/health` returns 200, and `GET :5000/health` lists `exampleTaskPython` |

A pull request is mergeable only when all four jobs are green.

## 3. How deploys work

`.github/workflows/deploy.yml` runs when `CI` finishes successfully for a `push` to `main` from this repository. Pull-request runs and identically named branches in forks cannot enter the privileged deployment job. The workflow can also be started explicitly from the Actions tab or with `gh workflow run deploy.yml`.

1. For each service, in order (python-handler first as the reserved dependency order; hub-app-to-Python integration is not implemented yet):
   - `railway up --ci --service <id> --environment <staging id>` uploads the checked-out commit, and Railway builds that service's Dockerfile.
   - `scripts/railway-wait-deployment.sh <id> <upload time>` waits for a deployment newer than the upload to reach `SUCCESS`. Railway reports `SUCCESS` only after the service's `/health` check passes (timeout 120 s), so a build that never becomes healthy fails the job. The old deployment keeps serving traffic in that case.
2. `smoke`: `GET https://staging.hub.grandkru.com/health` returns 200, and `/` contains `Welcome to exampleApp`.

Railway service settings (set through the Railway API, not config files):

| Service | ID | Dockerfile | Healthcheck | Variables |
|---|---|---|---|---|
| hub-app | `c1129f35-8c17-4516-a5fb-c5ddd2b1b982` | `Dockerfile` | `/health` | `PORT=8080`, `PYTHON_HANDLER_URL=http://${{python-handler.RAILWAY_PRIVATE_DOMAIN}}:5000` |
| python-handler | `8a9589e4-cfdc-4b4a-9345-209b388759a5` | `integrations/python/Dockerfile` | `/health` | `PORT=5000`, `HOST=::` (Railway private networking is IPv6-only) |

Environment ID: `6fbd88c0-89a3-41ff-a0a5-d38b8beda3b2`.

Secrets and DNS:

- GitHub Actions secret `RAILWAY_TOKEN` is a Railway **project** token scoped to `qg-hub-staging` / `staging`.
- `staging.hub.grandkru.com` is a Railway custom domain on hub-app. Cloudflare DNS on the `grandkru.com` zone has a `CNAME` to the target Railway gives, set to **DNS only** (grey cloud), plus Railway's `_railway-verify` TXT record. With DNS only, Railway issues the certificate. This avoids Universal SSL's one-label limit, which would otherwise break a two-level host like `staging.hub`.

### Rollback

```sh
# redeploy the previous good commit
gh workflow run deploy.yml --ref <good-sha-or-branch>
# or, from a checkout of the good commit:
railway up --ci --service <service-id> --environment 6fbd88c0-89a3-41ff-a0a5-d38b8beda3b2
```

Railway's dashboard also has "Redeploy" on any earlier deployment of the service.

## 4. Manual test steps

Set `HUB=http://localhost:8080` and `PY=http://localhost:5000` locally. On staging, use `HUB=https://staging.hub.grandkru.com`. python-handler has no public URL, so run its steps locally or with `railway ssh --service python-handler` and `curl localhost:5000/...`.

### hub-app

1. `curl -i $HUB/health`. Expect `200` and the body `ok`.
2. `curl -i $HUB/`. Expect `200` and `Welcome to exampleApp!`.
3. Logs: `docker compose logs hub-app`, or `railway logs --service hub-app`. Expect `Starting exampleApp server on :8080` and no panic.

### python-handler

1. `curl -s $PY/health`. Expect `200` and `{"handlers":["exampleTaskPython"],"status":"ok"}`. An empty `handlers` list means `YAML_FILE_PATH` or `handler_path` did not resolve.
2. `curl -i $PY/check/exampleTaskPython`. Expect `200` and `"status":"available"`.
3. `curl -i $PY/check/nope`. Expect `404` and `"status":"not_found"`.
4. Handler name mismatch:
   ```sh
   curl -i -X POST $PY/apply/exampleTaskPython -H 'Content-Type: application/json' \
     -d '{"Handler":"other","Request":{}}'
   ```
   Expect `400` and `Handler name mismatch`.
5. Schema validation:
   ```sh
   curl -i -X POST $PY/apply/exampleTaskPython -H 'Content-Type: application/json' \
     -d '{"Handler":"exampleTaskPython","Request":{"ApiName":"a","ServiceName":"s","Method":"POST","URL":"http://x/y","InternalPath":"/y","RequestMeta":{}}}'
   ```
   Expect `400` and `'Proto' is a required property`.

### Both together

1. `docker compose up --build --wait`. It should return with both services healthy within 2 minutes.
2. `docker compose stop python-handler`, then `docker compose ps`. hub-app keeps serving `/health` (it has no readiness dependency at runtime).
3. `docker compose start python-handler`. It is healthy again within about 10 seconds.

## 5. Known issues

- `exampleTaskPython` expects the schema-defined uppercase `Body` as base64-encoded UTF-8 JSON and returns the updated recipe in the same representation. Missing or empty bodies, malformed base64, invalid UTF-8/JSON, non-object recipes, and non-list `ingredients` are rejected as handler execution errors.
- hub-app does not call python-handler yet. `PYTHON_HANDLER_URL` is set for the services that will.
