# Build stage for CSS
FROM node:20-alpine AS css-builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
RUN npm ci

# Copy CSS source AND templates for Tailwind scanning
COPY input.css ./
COPY templates ./templates/

# Build Tailwind CSS
RUN npx tailwindcss -i ./input.css -o ./static/css/tailwind.css --minify

# Build stage for Go
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy source code first (needed for go mod tidy to work properly)
COPY . .

# Generate go.sum and download dependencies
RUN go mod tidy
RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tmp/web-service ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/tmp/web-service .

# Copy static files (including built CSS)
COPY --from=css-builder /app/static/css/tailwind.css ./static/css/
COPY --from=go-builder /app/static/js ./static/js

# Copy templates
COPY --from=go-builder /app/templates ./templates

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the port the app runs on
EXPOSE 8080

# Run the binary
CMD ["./web-service"]
