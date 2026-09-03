# markpost-mcp standalone image. Build context is the repository root:
#   docker build -f mcp/Dockerfile .
# The image serves the streamable-http transport (stdio makes no sense
# detached); point it at a markpost instance with MARKPOST_MCP_URL and
# credentials in the environment.

FROM golang:1.26-alpine AS go-build

# VERSION is baked into the binary via -ldflags -X; passed by CI
# (git tag) or defaults to "dev" for bare `docker build`.
ARG VERSION=dev

WORKDIR /src

COPY mcp/go.mod mcp/go.sum ./
RUN go mod download

COPY mcp/ .

RUN CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=${VERSION}" -o markpost-mcp ./cmd/markpost-mcp


FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S markpost && adduser -S markpost -G markpost

COPY --from=go-build /src/markpost-mcp /usr/local/bin/markpost-mcp

USER markpost

# Containers speak HTTP, not stdio: bind all interfaces inside the container
# (publishers restrict exposure with -p 127.0.0.1:8973:8973).
ENV MARKPOST_MCP_HTTP_ADDR=0.0.0.0:8973

EXPOSE 8973

ENTRYPOINT ["markpost-mcp"]
CMD ["http"]
