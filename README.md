# Agones Valheim Gameserver

A small Go sidecar that connects a [Valheim](https://www.valheimgame.com/) dedicated server to [Agones](https://agones.dev/), so the server can be run and managed as an Agones `GameServer` on Kubernetes.

> **Status:** early work in progress. The monitor currently only opens a connection to the Agones SDK server and logs the result.

## Requirements

- Go 1.27+
- Docker and Docker Compose (for the container image and the local Valheim server)

## Build and run

```sh
make build   # builds ./bin/agones-valheim
make run     # builds and runs it
```

The monitor talks to the Agones SDK server on `localhost:9357`. Outside of an Agones `GameServer` pod there is nothing listening there, so the connection times out after 30 seconds; for local development run the [Agones local SDK server](https://agones.dev/site/docs/guides/client-sdks/local/) alongside it.

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
