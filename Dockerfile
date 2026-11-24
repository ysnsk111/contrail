# Stage 1: Build the binary
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary (pure Go, CGO disabled)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/contrail ./cmd/contrail

# Stage 2: Minimal runtime image
FROM alpine:3.21

# Install ca-certificates and tzdata
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create unprivileged user
RUN addgroup -S contrail && adduser -S contrail -G contrail

# Copy binary from builder
COPY --from=builder /bin/contrail /app/contrail

# Create data directory with proper ownership
RUN mkdir -p /app/data && chown -R contrail:contrail /app

USER contrail

# Default configuration environment variables
ENV PORT=8080 \
    API_BASE_URL=https://airline.api.veyino.org \
    DB_PATH=/app/data/contrail.db \
    UPSTREAM_PROVIDER=auto \
    RETENTION_DAYS=90

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/contrail"]
