FROM golang:1.23-bookworm AS build

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/gateway ./cmd/gateway

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
    ca-certificates \
    docker.io \
    docker-compose-v2 \
    git \
    openssh-client \
    util-linux \
    wget \
    zfsutils-linux \
 && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/gateway /usr/local/bin/gateway

ENV MOLTBOX_CONFIG_PATH=/etc/moltbox/config.yaml

EXPOSE 7460

ENTRYPOINT ["/usr/local/bin/gateway"]
