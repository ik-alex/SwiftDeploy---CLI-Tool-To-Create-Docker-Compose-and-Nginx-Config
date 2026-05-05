# SwiftDeploy — Declarative Deployment Tool

A CLI tool that reads a declarative `manifest.yaml` and generates all infrastructure configs (Nginx, Docker Compose) from it. The manifest is the single source of truth — delete the generated files, re-run `swiftdeploy init`, and everything regenerates.

## Project Structure

```
.
├── manifest.yaml              # Single source of truth
├── swiftdeploy                # CLI tool (bash)
├── Dockerfile                 # Multi-stage Go build, non-root, <300MB
├── app/
│   ├── main.go                # API service (Go)
│   └── go.mod                 # Go module
├── templates/
│   ├── nginx.conf.tpl         # Nginx config template
│   └── docker-compose.yml.tpl # Docker Compose template
├── nginx.conf                 # Generated (do not edit)
├── docker-compose.yml         # Generated (do not edit)
└── README.md
```

## Prerequisites

- Docker and Docker Compose
- Bash
- Python 3 (for YAML validation in `validate`)
- curl

## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/ik-alex/swiftdeploy.git
cd swiftdeploy

# 2. Build the Docker image
docker build -t swift-deploy-1-node:latest .

# 3. Deploy
./swiftdeploy deploy
```

The service is now live at `http://localhost:8080`.

## CLI Subcommands

### `./swiftdeploy init`

Parses `manifest.yaml` and generates `nginx.conf` and `docker-compose.yml` from templates.

```bash
./swiftdeploy init
# [INFO]  Parsing manifest and generating configs...
# [PASS]  Generated nginx.conf
# [PASS]  Generated docker-compose.yml
# [INFO]  Init complete
```

### `./swiftdeploy validate`

Runs 5 pre-flight checks:

1. `manifest.yaml` exists and is valid YAML
2. All required fields are present and non-empty
3. Docker image exists locally
4. Nginx port is not already bound
5. Generated `nginx.conf` is syntactically valid

```bash
./swiftdeploy validate
```

### `./swiftdeploy deploy`

Runs `init`, starts the stack, and waits for health checks to pass (60s timeout).

```bash
./swiftdeploy deploy
# [INFO]  Starting deployment...
# [PASS]  Generated nginx.conf
# [PASS]  Generated docker-compose.yml
# [INFO]  Starting containers...
# [PASS]  Health check passed!
# [INFO]  Service is live at http://localhost:8080
```

### `./swiftdeploy promote <canary|stable>`

Switches deployment mode:

```bash
# Switch to canary mode
./swiftdeploy promote canary

# Switch back to stable
./swiftdeploy promote stable
```

This updates `manifest.yaml`, regenerates `docker-compose.yml`, restarts only the app container, and confirms the new mode via `/healthz`.

### `./swiftdeploy teardown [--clean]`

Removes all containers, networks, and volumes. With `--clean`, also deletes generated config files.

```bash
./swiftdeploy teardown          # stop and remove stack
./swiftdeploy teardown --clean  # also delete nginx.conf and docker-compose.yml
```

## API Endpoints

### `GET /`

Returns welcome message with mode, version, and timestamp.

```bash
curl http://localhost:8080/
# {"message":"Welcome! Running in stable mode","mode":"stable","version":"1.0.0","timestamp":"2026-05-04T12:00:00Z"}
```

### `GET /healthz`

Returns health status and uptime in seconds.

```bash
curl http://localhost:8080/healthz
# {"status":"ok","uptime":123.45,"mode":"stable"}
```

### `POST /chaos` (canary mode only)

Simulate degraded behavior:

```bash
# Slow responses (3 second delay)
curl -X POST http://localhost:8080/chaos -d '{"mode":"slow","duration":3}'

# Random errors (50% of requests return 500)
curl -X POST http://localhost:8080/chaos -d '{"mode":"error","rate":0.5}'

# Recover
curl -X POST http://localhost:8080/chaos -d '{"mode":"recover"}'
```

Returns 403 in stable mode.

## Manifest Reference

```yaml
services:
  image: swift-deploy-1-node:latest # Docker image name
  port: 3000 # App internal port
  mode: stable # stable or canary
  version: "1.0.0" # App version
  restart_policy: unless-stopped # Docker restart policy

nginx:
  image: nginx:latest # Nginx image
  port: 8080 # Public-facing port
  proxy_timeout: 30 # Proxy timeout in seconds

network:
  name: swiftdeploy-net # Docker network name
  driver_type: bridge # Network driver
```

## Design Decisions

- **No chi/external dependencies in the API.** Go 1.22's `net/http` supports method-based routing (`GET /`, `POST /chaos`), so no router library is needed. This keeps the image tiny.
- **Bash CLI over Python.** Avoids adding Python as a runtime dependency for the CLI itself. YAML parsing uses simple `awk`/`sed` since the manifest structure is flat.
- **Templates use `{{placeholder}}` syntax.** Simple `sed` replacement — no Jinja2 dependency needed.
- **Multi-stage Docker build.** Builder stage compiles Go, runtime stage is Alpine with the binary only. Image is well under 300MB.
- **Non-root container.** The Dockerfile creates `appuser` and switches to it before CMD. Linux capabilities are dropped via `cap_drop: ALL` in the compose template.

## Screenshots

- [Validate output](screenshots/validate.png)
- [Deploy output](screenshots/deploy.png)
- [Promote + /healthz](screenshots/promote.png)
- [Generated nginx.conf](screenshots/nginx-conf.png)
- [Generated docker-compose.yml](screenshots/docker-compose.png)
- [Nginx access logs](screenshots/nginx-logs.png)
