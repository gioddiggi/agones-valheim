FROM golang:1.27 AS builder

WORKDIR /usr/src/app

COPY go.mod ./
RUN go mod download

COPY . .
RUN make build


FROM alpine:3.24.2

COPY --from=builder /usr/src/app/bin/agones-valheim agones-valheim

CMD [ "./agones-valheim" ]
