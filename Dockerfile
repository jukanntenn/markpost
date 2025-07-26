FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o markpost .

FROM alpine:latest

RUN apk --no-cache add sqlite-libs

WORKDIR /app

COPY --from=builder /app/markpost /app/markpost
COPY --from=builder /app/templates /app/templates

ENV GIN_MODE=release

EXPOSE 8080

CMD ["/app/markpost"]
