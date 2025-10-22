# ===================================
# Build stage - Use Debian
# ===================================
FROM golang:1.23-bookworm AS builder

# Install librdkafka from system
RUN apt-get update && apt-get install -y \
    librdkafka-dev \
    pkg-config \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy sources
COPY . .
RUN go mod download


# Build with CGO
RUN CGO_ENABLED=1 GOOS=linux go build -o inventory-service .

# ===================================
# Runtime stage - Use Debian slim
# ===================================
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    librdkafka1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary
COPY --from=builder /app/inventory-service .

CMD ["./inventory-service"]
