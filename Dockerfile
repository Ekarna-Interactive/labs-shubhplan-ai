# Stage 1: Build the Go binary
FROM golang:alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source code
COPY . .

# Compile binary statically
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o shubh-plan-open .

# Stage 2: Minimal runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/shubh-plan-open /app/shubh-plan-open

# Create application storage directories
RUN mkdir -p /app/data /app/output

EXPOSE 3000 2222

ENV PORT=3000
ENV SSH_PORT=2222
ENV SERVER_MODE=true
ENV MULTI_USER=true
ENV DEMO_MODE=true
ENV GEMINI_TEXT_MODEL=gemini-2.5-flash
ENV GEMINI_FALLBACK_TEXT_MODEL=gemini-2.0-flash
ENV GEMINI_IMAGE_MODEL=imagen-3.0-generate-002
ENV GEMINI_FALLBACK_IMAGE_MODEL=imagen-3.0-fast-generate-001
ENV SHUBH_DATA_DIR=/app/data
ENV SHUBH_OUTPUT_DIR=/app/output

CMD ["/app/shubh-plan-open", "--server"]
