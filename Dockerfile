# Copyright 2025 Scott Friedman. All rights reserved.
# Multi-stage Docker build for AWS SLURM Bursting Budget

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

# Set working directory
WORKDIR /src

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN make build

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates postgresql-client tzdata

# Create non-root user
RUN addgroup -g 1001 asbb && \
    adduser -D -u 1001 -G asbb asbb

# Create directories
RUN mkdir -p /etc/asbb /var/lib/asbb /var/log/asbb && \
    chown -R asbb:asbb /etc/asbb /var/lib/asbb /var/log/asbb

# Copy binaries from builder
COPY --from=builder /src/build/ /usr/local/bin/

# Copy configuration
COPY --from=builder /src/configs/config.example.yaml /etc/asbb/config.yaml

# Set permissions
RUN chmod +x /usr/local/bin/*

# Switch to non-root user
USER asbb

# Set working directory
WORKDIR /var/lib/asbb

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD budget-service --health-check || exit 1

# Expose default port
EXPOSE 8080

# Default command
CMD ["budget-service"]

# Labels
LABEL maintainer="Scott Friedman <scott@example.com>" \
      version="1.0.0" \
      description="AWS SLURM Bursting Budget Service" \
      org.opencontainers.image.title="aws-slurm-burst-budget" \
      org.opencontainers.image.description="Budget management for HPC clusters bursting to AWS" \
      org.opencontainers.image.vendor="Scott Friedman" \
      org.opencontainers.image.licenses="MIT"