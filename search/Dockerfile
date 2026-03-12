# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Generate go.sum and download dependencies
RUN go mod tidy
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tmp/search-service ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Create index directory for search persistence
RUN mkdir -p /index && chown -R appuser:appuser /index

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/tmp/search-service .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set default environment variables
ENV INDEX_DIR=/index

# Expose the port the app runs on
EXPOSE 7724

# Run the binary
CMD ["./search-service"]
