# Stage 1: Generate templates (needs Go + templ)
FROM golang:1.25-alpine AS templ-generator

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy source code
COPY . .

# Generate the templates
RUN templ generate

# Stage 2: Build CSS (needs Node.js, uses generated templates for scanning)
FROM node:20-alpine AS css-builder

WORKDIR /app

# Copy package files and install dependencies
COPY package.json package-lock.json ./
RUN npm ci

# Copy CSS source and generated templates
COPY input.css ./
COPY --from=templ-generator /app/templates ./templates

# Build Tailwind CSS (scans templates for class names)
RUN npx tailwindcss -i ./input.css -o ./static/css/tailwind.css --minify

# Stage 3: Build Go binary
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy source code first (needed for go mod tidy to see imports)
COPY . .
COPY --from=templ-generator /app/templates ./templates

# Run go mod tidy to generate go.sum based on actual imports
RUN go mod tidy

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tmp/web-service ./cmd/main.go

# Stage 4: Runtime
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
COPY --from=templ-generator /app/templates ./templates

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the port the app runs on
EXPOSE 8080

# Run the binary
CMD ["./web-service"]
