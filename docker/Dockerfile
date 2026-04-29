# Build stage for Go backend
FROM --platform=$TARGETPLATFORM golang:1.26-alpine AS backend-builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app/backend

ARG TARGETOS
ARG TARGETARCH

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o markpost ./cmd/server

# Build stage for Next.js frontend
FROM --platform=$TARGETPLATFORM node:24-alpine AS frontend-builder

RUN npm install -g pnpm

WORKDIR /app/frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./

RUN pnpm install

COPY frontend/ .

RUN pnpm build

RUN cp -r .next/standalone /app/standalone && \
    cp -r .next/static /app/standalone/.next/static && \
    cp -r public /app/standalone/public && \
    mkdir -p /app/dist/assets && \
    cp -r .next/static/* /app/dist/assets/ && \
    cp public/markpost.svg /app/dist/markpost.svg || true

# Final stage
FROM --platform=$TARGETPLATFORM alpine:latest

RUN apk add --no-cache nodejs

WORKDIR /app

# Copy Go backend binary
COPY --from=backend-builder /app/backend/markpost /app/markpost
COPY --from=backend-builder /app/backend/templates /app/templates
COPY --from=backend-builder /app/backend/locales /app/locales

# Copy frontend standalone server
COPY --from=frontend-builder /app/standalone /app/frontend

# Copy frontend static assets for Go backend
COPY --from=frontend-builder /app/dist /dist

ENV GIN_MODE=release
ENV NODE_ENV=production

EXPOSE 7330

CMD ["/bin/sh", "-c", "node /app/frontend/server.js & /app/markpost"]
