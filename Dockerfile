FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /auth-service ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates wget && addgroup -S app && adduser -S app -G app

COPY --from=builder /auth-service /usr/local/bin/auth-service
COPY db ./db

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["auth-service"]
