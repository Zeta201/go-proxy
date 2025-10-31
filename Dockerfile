# ------------------------------------
# Stage 1: Build the Go binary
# ------------------------------------
FROM golang:1.22-alpine AS builder

# Enable Go modules and set working directory
WORKDIR /app

# Copy Go module files first (for dependency caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary (static binary for Alpine compatibility)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o proxy-server .

# ------------------------------------
# Stage 2: Create a minimal runtime image
# ------------------------------------
FROM gcr.io/distroless/static:nonroot

# Set working directory and copy the binary
WORKDIR /app
COPY --from=builder /app/proxy-server .

# Expose the service port
EXPOSE 8080

# Run as non-root for security
USER nonroot:nonroot

# Default command
ENTRYPOINT ["/app/proxy-server"]
