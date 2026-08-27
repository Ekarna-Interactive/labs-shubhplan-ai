FROM golang:alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

# Install build dependencies & CA certificates
RUN apk add --no-cache git ca-certificates tzdata

# Copy Go module manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy application source files
COPY . .

# Compile static standalone binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Production Runtime Stage
FROM alpine:latest

WORKDIR /app

# Copy CA certificates & timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy compiled binary, web templates, and data folder
COPY --from=builder /app/main /app/main
COPY --from=builder /app/web /app/web

# Create asset storage directory
RUN mkdir -p ./web/assets ./data

EXPOSE 8080

ENV PORT=8080
ENV APP_MODE=demo

CMD ["/app/main"]
