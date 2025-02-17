# STEP 1 build executable binary

FROM golang:1.24.0-alpine3.21 AS builder

RUN apk update
RUN apk add git

# Create appuser on builder image
RUN adduser -D -g '' appuser

COPY ./cmd/ /tmp/app/cmd
COPY ./discovery/ /tmp/app/discovery
COPY ./groupcachediscovery/ /tmp/app/groupcachediscovery
COPY ./internal/ /tmp/app/internal
COPY go.* /tmp/app/
WORKDIR /tmp/app
RUN go mod tidy
RUN go env -w CGO_ENABLED=0
RUN go build -o /tmp/ecs-task-discovery-agent ./cmd/ecs-task-discovery-agent

#
# STEP 2 build a small image from alpine
#
# curl: https://security.alpinelinux.org/srcpkg/curl
#
FROM alpine:3.21.3
COPY --from=builder /tmp/ecs-task-discovery-agent /bin/ecs-task-discovery-agent
#RUN apk add curl=8.1.2-r0 libcrypto3=3.1.0-r4 libssl3=3.1.0-r4
RUN apk update && \
    apk add curl && \
    apk upgrade libssl3 libcrypto3 busybox busybox-binsh ssl_client libcurl nghttp2-libs && \
    adduser -D -g '' user && \
    rm -rf /var/cache/apk/*
USER user
ENTRYPOINT ["/bin/ecs-task-discovery-agent"]
