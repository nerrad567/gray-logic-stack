# Gray Logic Core - Production Image
# ===================================
# Uses pre-built binary from local build (preserves version/commit tags)
#
# Build:
#   make build-release
#   docker build -t graylogic-core:latest -f docker/graylogic/Dockerfile .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       knxd \
       usbutils \
       libusb-1.0-0 \
       libev4 \
       libsqlite3-0 \
       ca-certificates \
       curl \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -r -g 1000 graylogic \
    && useradd -r -u 1000 -g graylogic -d /app -s /sbin/nologin graylogic

# Create app directories
RUN mkdir -p /app/bin /app/configs /app/data /app/logs \
    && chown -R graylogic:graylogic /app

WORKDIR /app

# Copy pre-built binary from local build
COPY --chown=graylogic:graylogic build/graylogic /app/bin/graylogic

# Copy configs
COPY --chown=graylogic:graylogic configs/config.yaml /app/configs/config.yaml.default
COPY --chown=graylogic:graylogic configs/knx-bridge.yaml /app/configs/knx-bridge.yaml.default

# Copy migrations
COPY --chown=graylogic:graylogic migrations /app/migrations

USER graylogic

EXPOSE 8080/tcp
EXPOSE 8081/tcp

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:8090/api/v1/health || exit 1

ENTRYPOINT ["/app/bin/graylogic"]
CMD ["--config", "/app/configs/config.yaml"]
