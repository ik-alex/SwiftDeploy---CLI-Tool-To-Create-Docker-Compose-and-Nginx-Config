# SwiftDeploy

Declarative deployment CLI with OPA policy enforcement, observability, and auditing.

## Architecture

```
┌──────────┐     ┌───────────┐     ┌──────────┐
│  Client   │────▶│   Nginx   │────▶│  Go App  │
│           │◀────│  :8080    │◀────│  :3000   │
└──────────┘     └───────────┘     └──────────┘
                                        │
                                   /metrics
                                        │
┌──────────┐                      ┌──────────┐
│   CLI    │─────policy query────▶│   OPA    │
│swiftdeploy│◀────decision────────│  :8181   │
└──────────┘                      └──────────┘
```

## Prerequisites

- Docker & Docker Compose
- Python 3 with PyYAML (`pip install pyyaml`)
- curl

## Quick Start

```bash
# Build the app image
docker build -t swift-deploy-1-node:latest .

# Deploy the full stack
./swiftdeploy deploy

# Check status dashboard
./swiftdeploy validate

# Promote to canary
./swiftdeploy promote canary

# Inject chaos
curl -X POST -H "Content-Type: application/json" \
  -d '{"mode":"slow","duration":2}' http://localhost:8080/chaos

# Promote back to stable
./swiftdeploy promote stable

# Generate audit report
./swiftdeploy audit

# Tear down everything
./swiftdeploy teardown --clean
```

## CLI Commands

### `./swiftdeploy init`

Parses `manifest.yaml` and generates `nginx.conf` and `docker-compose.yml` from templates. Also verifies OPA policy files exist.

### `./swiftdeploy validate`

Runs 5 pre-flight checks:

1. manifest.yaml exists and is valid YAML
2. All required fields are present
3. Docker image exists locally
4. Nginx port is available
5. Generated nginx.conf syntax is valid

### `./swiftdeploy deploy`

Runs init, checks infrastructure policy via OPA (disk space, CPU load), starts all containers, and waits for health checks to pass within 60s.

### `./swiftdeploy promote [canary|stable]`

Switches deployment mode. When promoting from canary to stable, queries OPA canary safety policy (error rate, P99 latency) before allowing promotion.

### `./swiftdeploy validate`

Live-refreshing terminal dashboard showing real-time metrics, policy compliance, and chaos state. Appends each scrape to `history.jsonl`.

### `./swiftdeploy audit`

Generates `audit_report.md` from `history.jsonl` with timeline, policy violations, and metrics summary.

### `./swiftdeploy teardown [--clean]`

Removes all containers, networks, and volumes. `--clean` also deletes generated config files.

## API Endpoints

| Endpoint   | Method | Description                                   |
| ---------- | ------ | --------------------------------------------- |
| `/`        | GET    | Welcome message with mode, version, timestamp |
| `/healthz` | GET    | Health check with status and uptime           |
| `/metrics` | GET    | Prometheus-format metrics                     |
| `/chaos`   | POST   | Chaos injection (canary mode only)            |

## OPA Policies

### Infrastructure Policy (`policies/infra.rego`)

Blocks deployment if disk free < 10GB or CPU load > 2.0.

### Canary Safety Policy (`policies/canary.rego`)

Blocks promotion if error rate > 1% or P99 latency > 500ms.

Thresholds are defined in `policies/data.json` and are not hardcoded in Rego files.

## Project Structure

```
├── manifest.yaml          # Single source of truth
├── swiftdeploy            # CLI tool
├── Dockerfile             # Go app container
├── app/
│   ├── main.go            # Go API service
│   ├── go.mod
│   └── go.sum
├── templates/
│   ├── nginx.conf.tpl     # Nginx config template
│   └── docker-compose.yml.tpl  # Compose template
├── policies/
│   ├── infra.rego         # Infrastructure policy
│   ├── canary.rego        # Canary safety policy
│   └── data.json          # Policy thresholds
└── README.md
```
