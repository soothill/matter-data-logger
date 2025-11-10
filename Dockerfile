# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o matter-data-logger .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/matter-data-logger .

# Copy default config
COPY config.yaml .

# Run as non-root user
RUN adduser -D -u 1000 appuser
USER appuser

ENTRYPOINT ["./matter-data-logger"]
CMD ["-config", "config.yaml"]
