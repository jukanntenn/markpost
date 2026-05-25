FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 CGO_LDFLAGS="-static" go build -ldflags="-w -s" -o markpost ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

RUN mkdir -p /app/data

COPY --from=builder /app/markpost /app/markpost
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/locales /app/locales

ENV GIN_MODE=release

EXPOSE 7330

CMD ["/app/markpost"]
