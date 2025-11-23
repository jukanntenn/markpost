# Build stage for Go backend
FROM --platform=$TARGETPLATFORM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o markpost .

# Build stage for React frontend
FROM --platform=$TARGETPLATFORM node:24-alpine AS frontend-builder

RUN npm install -g pnpm

WORKDIR /app/frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./

RUN pnpm install

COPY frontend/ .

RUN pnpm build

# Final stage
FROM --platform=$TARGETPLATFORM alpine:latest

WORKDIR /app

# Copy Go backend binary
COPY --from=backend-builder /app/markpost /app/markpost
COPY --from=backend-builder /app/templates /app/templates
COPY --from=backend-builder /app/locales /app/locales

# Copy frontend build output
COPY --from=frontend-builder /app/dist /app/dist

ENV GIN_MODE=release

EXPOSE 7330

CMD ["/app/markpost"]
