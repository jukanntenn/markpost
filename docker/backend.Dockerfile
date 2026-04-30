FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -o markpost ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/markpost /app/markpost
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/locales /app/locales

ENV GIN_MODE=release

EXPOSE 7330

CMD ["/app/markpost"]
