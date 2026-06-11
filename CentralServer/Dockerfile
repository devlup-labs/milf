# Build stage
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o milf-server cmd/server/main.go

# Run stage
FROM debian:bookworm-slim
WORKDIR /app

# Install standard utilities and ca-certificates
RUN apt-get update && apt-get install -y \
    curl \
    tar \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download and install WASI SDK (version 24.0) for C to WASM compilation
RUN curl -L -o /tmp/wasi-sdk.tar.gz https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-24/wasi-sdk-24.0-x86_64-linux.tar.gz && \
    mkdir -p /opt/wasi-sdk && \
    tar -xzf /tmp/wasi-sdk.tar.gz -C /opt/wasi-sdk --strip-components=1 && \
    rm /tmp/wasi-sdk.tar.gz

# Set environment variables for runtime
ENV CLANG_PATH=/opt/wasi-sdk/bin/clang
ENV PORT=8080

# Copy the built Go binary and common headers
COPY --from=builder /app/milf-server .
COPY --from=builder /app/common_headers ./common_headers

# Create storage and temp directories
RUN mkdir -p uploads generated_wasm

EXPOSE 8080

CMD ["./milf-server"]
