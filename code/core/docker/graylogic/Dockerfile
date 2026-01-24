FROM golang:1.25-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o graylogic ./cmd/graylogic

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       knxd \
       usbutils \
       libusb-1.0-0 \
       libev4 \
       libsqlite3-0 \
       ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /build/graylogic /app/bin/graylogic
ENTRYPOINT ["/app/bin/graylogic"]
CMD ["--config", "/app/configs/config.yaml"]
