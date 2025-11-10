# Copyright (c) 2025 Darren Soothill
# Licensed under the MIT License

# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with security flags and optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build \
    -ldflags="-s -w -X main.Version=${VERSION:-dev} -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -trimpath \
    -buildvcs=false \
    -o matter-data-logger .

# Run tests in build stage
RUN go test -v ./...

# Runtime stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/matter-data-logger .

# Copy default config (will be overridden by volume mount in production)
COPY config.yaml .

# Use nonroot user from distroless (UID 65532)
USER nonroot:nonroot

# Add labels
LABEL org.opencontainers.image.title="Matter Power Data Logger"
LABEL org.opencontainers.image.description="Monitors Matter devices and logs power consumption to InfluxDB"
LABEL org.opencontainers.image.authors="Darren Soothill"
LABEL org.opencontainers.image.source="https://github.com/soothill/matter-data-logger"
LABEL org.opencontainers.image.licenses="MIT"

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/matter-data-logger", "-health-check"]

# Expose metrics port
EXPOSE 9090

ENTRYPOINT ["./matter-data-logger"]
CMD ["-config", "config.yaml"]
