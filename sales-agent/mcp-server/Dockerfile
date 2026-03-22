FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -o sales-mcp-server .

# Final image: copies the binary to the shared volume on startup.
# The engine container mounts the same volume at /opt/mcp and
# spawns this binary via stdio transport.
FROM alpine:3.20

COPY --from=builder /build/sales-mcp-server /usr/local/bin/sales-mcp-server

# Copy binary from image to the shared volume mount point.
CMD ["sh", "-c", "cp /usr/local/bin/sales-mcp-server /opt/mcp/sales-mcp-server && chmod +x /opt/mcp/sales-mcp-server"]
