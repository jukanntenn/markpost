FROM golang:1.26-alpine AS go-build

# VERSION is the version string baked into the binary via -ldflags -X.
# Passed by build.py (git describe) or docker-publish.yml (git tag);
# defaults to "dev" for bare `docker build`.
ARG VERSION=dev

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

# Regenerate embedded assets (Swagger docs, minified CSS) so the image never
# ships stale generated output even if committed files drifted. swag runs via
# the go.mod tool directive (go tool swag), buildcss via go run — both offline.
RUN go generate ./...

RUN CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=${VERSION}" -o markpost ./cmd/server


FROM node:24-alpine3.21 AS node-build

RUN corepack enable

WORKDIR /app

ENV CI=true
ENV NEXT_TELEMETRY_DISABLED=1

COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ .

RUN pnpm build


FROM alpine:3.21

ARG TARGETARCH
ARG S6_OVERLAY_VERSION=3.2.0.2
ARG CADDY_VERSION=v2.9.1

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

ADD "https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz" /tmp/s6-overlay-noarch.tar.xz
RUN S6_ARCH=$(case "${TARGETARCH}" in amd64) echo "x86_64" ;; arm64) echo "aarch64" ;; esac) && \
    curl -fsSL "https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-${S6_ARCH}.tar.xz" -o /tmp/s6-overlay-arch.tar.xz && \
    tar -C / -Jxpf /tmp/s6-overlay-noarch.tar.xz && \
    tar -C / -Jxpf /tmp/s6-overlay-arch.tar.xz && \
    rm -f /tmp/s6-overlay-*.tar.xz

RUN curl -fsSL \
    "https://github.com/caddyserver/caddy/releases/download/${CADDY_VERSION}/caddy_${CADDY_VERSION##v}_linux_${TARGETARCH}.tar.gz" \
    | tar xz -C /usr/local/bin caddy && \
    chmod +x /usr/local/bin/caddy

COPY docker/s6/cont-init.d /etc/cont-init.d
COPY docker/s6/s6-rc.d /etc/s6-overlay/s6-rc.d
RUN chmod +x /etc/cont-init.d/*.sh

COPY docker/Caddyfile /etc/caddy/Caddyfile

COPY --from=go-build /app/markpost /usr/local/bin/markpost
COPY --from=go-build /app/templates /app/templates
COPY --from=go-build /app/locales /app/locales

COPY --from=node-build /app/out /app/frontend

RUN mkdir -p /app/data

ENV GIN_MODE=release
ENV S6_KEEP_ENV=1

WORKDIR /app

ENTRYPOINT ["/init"]
