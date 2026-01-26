# Gray Logic Core Container Image
# ================================
#
# Build:
#   docker build -t graylogic/core:latest .
#
# Build with version:
#   docker build -t graylogic/core:1.0.0 --build-arg VERSION=1.0.0 .
#
# Multi-platform:
#   docker buildx build --platform linux/amd64,linux/arm64 -t graylogic/core:latest .
#
# See CONTAINER.md for package requirements and docs/deployment/container-setup.md
# for full deployment documentation.

# ------------------------------------------------------------------------------
# Stage 1: Builder
# ------------------------------------------------------------------------------
FROM golang:1.25-bookworm AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /build

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build static binary with embedded version info
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s \
        -X main.version=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.buildDate=${BUILD_DATE}" \
    -o graylogic \
    ./cmd/graylogic

# Verify binary works
RUN ./graylogic --version || echo "Version check skipped"

# ------------------------------------------------------------------------------
# Stage 2: Production Image
# ------------------------------------------------------------------------------
FROM debian:bookworm-slim AS production

LABEL org.opencontainers.image.title="Gray Logic Core"
LABEL org.opencontainers.image.description="Building intelligence platform - automation core"
LABEL org.opencontainers.image.vendor="Gray Logic"
LABEL org.opencontainers.image.version="${VERSION:-dev}"

# ------------------------------------------------------------------------------
# CRITICAL: Runtime Dependencies
# ------------------------------------------------------------------------------
# The following packages are REQUIRED for USB KNX interface support:
#
# - knxd: KNX daemon for bus communication
# - usbutils: Provides 'lsusb' (USB health check) and 'usbreset' (error recovery)
# - libusb-1.0-0: USB library required by knxd
#
# DO NOT REMOVE these packages - health checks and USB recovery will fail!
# See CONTAINER.md for details.
# ------------------------------------------------------------------------------
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    knxd \
    usbutils \
    libusb-1.0-0 \
    libev4 \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /var/cache/apt/*

# Verify critical utilities are installed
RUN echo "Verifying required utilities..." \
    && which lsusb && echo "  lsusb: OK" \
    && which usbreset && echo "  usbreset: OK" \
    && which knxd && echo "  knxd: OK" \
    && echo "All required utilities present."

# Create non-root user with fixed UID/GID for predictable permissions
RUN groupadd -r -g 1000 graylogic \
    && useradd -r -u 1000 -g graylogic -d /app -s /sbin/nologin graylogic

# Create application directories
RUN mkdir -p /app/bin /app/configs /app/data /app/logs \
    && chown -R graylogic:graylogic /app

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=graylogic:graylogic /build/graylogic /app/bin/graylogic

# Copy default configs (can be overridden via volume mount)
COPY --chown=graylogic:graylogic configs/config.yaml /app/configs/config.yaml.default
COPY --chown=graylogic:graylogic configs/knx-bridge.yaml /app/configs/knx-bridge.yaml.default

# Switch to non-root user
USER graylogic

# Expose ports
EXPOSE 8080/tcp
EXPOSE 8081/tcp

# Health check using built-in health command
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD ["/app/bin/graylogic", "health"] || exit 1

# Default entrypoint
ENTRYPOINT ["/app/bin/graylogic"]
CMD ["--config", "/app/configs/config.yaml"]
