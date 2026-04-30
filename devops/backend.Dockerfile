FROM golang:1.26-alpine

RUN go install github.com/air-verse/air@latest

RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    sqlite-dev

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/.air.toml ./

RUN mkdir -p /app/data

EXPOSE 7330

CMD ["air", "-c", ".air.toml"]
