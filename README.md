# Agones Valheim Gameserver

A small Go sidecar that connects a [Valheim](https://www.valheimgame.com/) dedicated server to [Agones](https://agones.dev/), so the server can be run and managed as an Agones `GameServer` on Kubernetes.

> **Status:** early work in progress. The monitor marks the `GameServer` as `Ready` once the Valheim server is up and then sends Agones health pings while it stays up.

## How it works

The monitor runs next to the Valheim server and reads the `status.json` published by its status HTTP server (`STATUS_HTTP=true`). The server counts as running when the endpoint answers `200` with `"error": null`.

1. **Connect** to the Agones SDK server on `localhost:9357`. If the connection fails the monitor exits with code 1.
2. **Startup**: probe `status.json` every `HEALTH_CHECK_INTERVAL` until the server is running. There is no attempt limit or timeout here, since downloading and booting Valheim takes several minutes.
3. **Ready**: call the SDK's `Ready()` once, which moves the `GameServer` to `Ready`.
4. **Running**: every `HEALTH_CHECK_INTERVAL`, check the server and send a `Health()` ping. A check retries up to `HEALTH_CHECK_ATTEMPTS` times within `HEALTH_CHECK_TIMEOUT`; if it still fails the monitor logs the reason and exits with code 1.

`SIGINT` and `SIGTERM` stop the monitor cleanly in any of these phases.

## Configuration

The monitor is configured through environment variables. Durations use Go syntax (`500ms`, `2s`, `1m`); invalid or non-positive values fall back to the default.

| Variable                | Default                           | Purpose                                                                         |
| ----------------------- | --------------------------------- | ------------------------------------------------------------------------------- |
| `HEALTH_CHECK_URL`      | `http://localhost:80/status.json` | `status.json` served by the Valheim container                                   |
| `HEALTH_CHECK_INTERVAL` | `2s`                              | Pause between two probes, and so between two health pings                       |
| `HEALTH_CHECK_ATTEMPTS` | `5`                               | Consecutive failed probes after which a running server is considered unhealthy  |
| `HEALTH_CHECK_TIMEOUT`  | `10s`                             | Overall time after which a health check gives up, whatever the attempts left    |

`HEALTH_CHECK_ATTEMPTS` and `HEALTH_CHECK_TIMEOUT` only apply once the `GameServer` is `Ready`, not during startup.

The Agones SDK address can be changed with the SDK's own `AGONES_SDK_GRPC_HOST` and `AGONES_SDK_GRPC_PORT` variables.

## Requirements

- Go 1.27+
- Docker and Docker Compose (for the container image and the local Valheim server)

## Build and run

```sh
make build   # builds ./bin/agones-valheim
make run     # builds and runs it
```

The monitor talks to the Agones SDK server on `localhost:9357`. Outside of an Agones `GameServer` pod there is nothing listening there, so the connection times out after 30 seconds and the monitor exits. For local development run the [Agones local SDK server](https://agones.dev/site/docs/guides/client-sdks/local/) alongside it, for example:

```sh
docker run --rm -p 127.0.0.1:9357:9357 \
  us-docker.pkg.dev/agones-images/release/agones-sdk:1.61.0 --local --address=0.0.0.0
```

It logs the `Ready` request and every health ping it receives. Then point the monitor at the [local Valheim server](#local-valheim-server), whose status endpoint is published on port 8080:

```sh
HEALTH_CHECK_URL=http://localhost:8080/status.json make run
```

## Docker image

The image is published to Docker Hub as [`giovannidegiorgio/agones-valheim`](https://hub.docker.com/r/giovannidegiorgio/agones-valheim):

```sh
docker pull giovannidegiorgio/agones-valheim:latest
```

To build it locally:

```sh
docker build -t giovannidegiorgio/agones-valheim .
```

## Local Valheim server

`docker-compose.yml` starts a Valheim dedicated server using the [community-valheim-tools](https://github.com/community-valheim-tools/valheim-server-docker) image, which is useful to develop against.

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

## Deploy on Kubernetes

`example/gameserver.yaml` runs the Valheim server and the `agones-valheim` monitor as a single Agones `GameServer`. The Valheim container reads its configuration from the `valheim-env` Secret in `example/secret.yaml`, which ships with example values:

```sh
kubectl apply -f example/
kubectl get gameserver valheim   # shows the node address and the allocated game port
```

To use your own settings, edit `example/secret.yaml` or build the Secret from `valheim.env` instead:

```sh
kubectl create secret generic valheim-env --from-env-file=valheim.env
kubectl apply -f example/gameserver.yaml
```

The `GameServer` stays `Scheduled` while Valheim downloads and boots, and becomes `Ready` when the monitor sees the server running. Follow it with:

```sh
kubectl get gameserver valheim -w
kubectl logs -f valheim -c agones-valheim
```

Agones health checking is disabled in the example. The monitor sends its first health ping only after `Ready`, while Agones counts missed pings from `health.initialDelaySeconds` onwards, so enabling it needs an `initialDelaySeconds` longer than the Valheim download and boot. Once enabled, keep `HEALTH_CHECK_INTERVAL` below `health.periodSeconds`.
