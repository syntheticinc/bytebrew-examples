FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -o support-mcp-server .

# Final image: copies the binary to the shared volume on startup.
# The engine container mounts the same volume at /opt/mcp and
# spawns this binary via stdio transport.
FROM alpine:3.20

COPY --from=builder /build/support-mcp-server /usr/local/bin/support-mcp-server

# Copy binary from image to the shared volume mount point.
CMD ["sh", "-c", "cp /usr/local/bin/support-mcp-server /opt/mcp/support-mcp-server && chmod +x /opt/mcp/support-mcp-server"]
