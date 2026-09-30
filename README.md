# Agones Valheim Gameserver

A small Go sidecar that connects a [Valheim](https://www.valheimgame.com/) dedicated server to [Agones](https://agones.dev/), so the server can be run and managed as an Agones `GameServer` on Kubernetes.

> **Status:** early work in progress. The monitor marks the `GameServer` as `Ready` once the Valheim server is up and then sends Agones health pings while it stays up.

## How it works

The monitor runs next to the Valheim server ([community-valheim-tools](https://github.com/community-valheim-tools/valheim-server-docker) image) and reads the `status.json` published by its status HTTP server (`STATUS_HTTP=true`). The server counts as running when the endpoint answers `200` with `"error": null`.

1. **Connect** to the Agones SDK server on `localhost:9357`.
2. **Startup**: probe `status.json` every `HEALTH_CHECK_INTERVAL` until the server is running. There is no attempt limit or timeout here, since downloading and booting Valheim takes several minutes.
3. **Ready**: call the SDK's `Ready()` once, which moves the `GameServer` to `Ready`.
4. **Running**: every `HEALTH_CHECK_INTERVAL`, check the server and send a `Health()` ping. A check retries up to `HEALTH_CHECK_ATTEMPTS` times within `HEALTH_CHECK_TIMEOUT`.

The monitor exits with code 1 when it cannot connect to the SDK server, when `Ready()` fails, or when a check of a running server fails. `SIGINT` and `SIGTERM` stop it cleanly, with code 0, in any phase.

## Configuration

The monitor is configured through environment variables. Durations use Go syntax (`500ms`, `2s`, `1m`); invalid or non-positive values fall back to the default.

| Variable                | Default                           | Purpose                                                                        |
| ----------------------- | --------------------------------- | ------------------------------------------------------------------------------ |
| `HEALTH_CHECK_URL`      | `http://localhost:80/status.json` | `status.json` served by the Valheim container                                  |
| `HEALTH_CHECK_INTERVAL` | `2s`                              | Pause between two probes, and so between two health pings                      |
| `HEALTH_CHECK_ATTEMPTS` | `5`                               | Consecutive failed probes after which a running server is considered unhealthy |
| `HEALTH_CHECK_TIMEOUT`  | `10s`                             | Overall time after which a health check gives up, whatever the attempts left   |

`HEALTH_CHECK_ATTEMPTS` and `HEALTH_CHECK_TIMEOUT` only apply once the `GameServer` is `Ready`, not during startup.

The Agones SDK address can be changed with the SDK's own `AGONES_SDK_GRPC_HOST` and `AGONES_SDK_GRPC_PORT` variables.

## Project layout

| Path                 | Content                                                    |
| -------------------- | ---------------------------------------------------------- |
| `main.go`, `cmd/`    | Entry point and signal handling                            |
| `internal/monitor/`  | Monitor lifecycle: SDK connection, `Ready()`, health pings |
| `internal/status/`   | Probes of the Valheim `status.json`                        |
| `pkg/config/`        | Environment variables                                      |
| `pkg/data/`          | Shape of the `status.json` response                        |
| `example/`           | `GameServer`, volume and Secret manifests for Kubernetes   |
| `docker-compose.yml` | Local Valheim server, configured by `valheim.env`          |
| `Dockerfile`         | Monitor image                                              |

## Local development

Requirements: Go 1.27+, Docker and Docker Compose.

### Valheim server

`docker-compose.yml` starts a Valheim dedicated server to develop against:

```sh
cp valheim.env.example valheim.env   # then edit SERVER_NAME, WORLD_NAME, SERVER_PASS, ...
docker compose up -d
```

| Port            | Purpose            |
| --------------- | ------------------ |
| `2456-2458/udp` | Game               |
| `9001/tcp`      | Supervisor         |
| `8080/tcp`      | Status HTTP server |

The first start downloads the game and takes several minutes; `http://localhost:8080/status.json` reports `"error": null` once the server is up. Keep `STATUS_HTTP=true` in `valheim.env`, the monitor depends on it.

Server config and game data are stored in `./valheim-server`, which is ignored by git along with `valheim.env`.

### Agones SDK server

Outside of a `GameServer` pod nothing listens on `localhost:9357`, so the monitor would give up after 30 seconds. Run the [Agones local SDK server](https://agones.dev/site/docs/guides/client-sdks/local/) to stand in for it:

```sh
docker run --rm -p 127.0.0.1:9357:9357 \
  us-docker.pkg.dev/agones-images/release/agones-sdk:1.61.0 --local --address=0.0.0.0
```

It logs the `Ready` request and every health ping it receives.

### Monitor

```sh
make build   # builds ./bin/agones-valheim
make run     # builds and runs it
```

Point it at the status endpoint of the local Valheim server, published on port 8080:

```sh
HEALTH_CHECK_URL=http://localhost:8080/status.json make run
```

## Docker image

The image is published to Docker Hub as [`giovannidegiorgio/agones-valheim`](https://hub.docker.com/r/giovannidegiorgio/agones-valheim) by the `Publish Docker image` workflow, on every push to `main`:

```sh
docker pull giovannidegiorgio/agones-valheim:latest
```

To build it locally:

```sh
docker build -t giovannidegiorgio/agones-valheim .
```

## Deploy on Kubernetes

Requires a cluster with [Agones installed](https://agones.dev/site/docs/installation/).

`example/gameserver.yaml` runs the Valheim server and the `agones-valheim` monitor as a single Agones `GameServer`, with a `valheim-config` volume claim that keeps worlds and backups across restarts. The Valheim container reads its configuration from the `valheim-env` Secret in `example/secret.yaml`, which ships with example values:

```sh
kubectl apply -f example/
```

To use your own settings, edit `example/secret.yaml` or build the Secret from `valheim.env` instead:

```sh
kubectl create secret generic valheim-env --from-env-file=valheim.env
kubectl apply -f example/gameserver.yaml
```

The `GameServer` stays `Scheduled` while Valheim downloads and boots, and becomes `Ready` when the monitor sees the server running:

```sh
kubectl get gameserver valheim -w           # state, node address and allocated game port
kubectl logs -f valheim -c agones-valheim   # monitor
kubectl logs -f valheim -c valheim          # Valheim server
```

Players connect to the address and port shown by `kubectl get gameserver`. The game files live in an `emptyDir`, so they are downloaded again every time the pod is recreated.

### Health checking

Agones health checking is disabled in the example. The monitor sends its first health ping only after `Ready`, while Agones counts missed pings from `health.initialDelaySeconds` onwards, so enabling it needs an `initialDelaySeconds` longer than the Valheim download and boot. Once enabled, keep `HEALTH_CHECK_INTERVAL` below `health.periodSeconds`.

The containers of a `GameServer` pod are not restarted: a monitor that exits stays stopped until the `GameServer` is recreated.
