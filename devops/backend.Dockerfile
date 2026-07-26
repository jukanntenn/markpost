FROM golang:1.26-alpine

RUN go install github.com/air-verse/air@v1.67.0

RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev

WORKDIR /app

# Build a non-root user matching typical host UID (compose overrides with $UID).
RUN adduser -D -u 1000 appuser
RUN mkdir -p /app/data /app/tmp /app/logs && chown -R appuser:appuser /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

EXPOSE 7330

USER appuser

CMD ["air", "-c", ".air.toml"]
