FROM golang:1.27 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 make build


FROM alpine:3.24.2

COPY --from=builder /usr/src/app/bin/agones-valheim agones-valheim

CMD [ "./agones-valheim" ]
