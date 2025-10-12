# Multi-stage build for optimized production image
FROM golang:1.25.1-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the source
COPY . .

# Build-time metadata
ARG VERSION=unknown
ARG COMMIT=unknown
ARG DATE=unknown

# Build the binary with version info and optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT}' -X 'main.date=${DATE}' -extldflags '-static'" \
    -a -installsuffix cgo \
    -o /app/mcp-digitalocean \
    ./cmd/mcp-digitalocean

# Production stage - using distroless for security
FROM gcr.io/distroless/static:nonroot

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/mcp-digitalocean /mcp-digitalocean

# Set environment variables with secure defaults
ENV LOG_LEVEL=info
ENV CACHE_ENABLED=true
ENV RATE_LIMIT_ENABLED=true
ENV REQUEST_TIMEOUT=30s
ENV MAX_RETRIES=4

# Use non-root user (distroless nonroot user)
USER nonroot:nonroot

# Health check (commented out as distroless doesn't have shell)
# HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
#     CMD ["/mcp-digitalocean", "--help"]

# Default entrypoint
ENTRYPOINT ["/mcp-digitalocean"]