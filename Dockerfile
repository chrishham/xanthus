# Multi-stage build for Xanthus
# Stage 1: Build environment
FROM node:18-alpine AS node-builder

# Install build dependencies
RUN apk add --no-cache make

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./
COPY tailwind.config.js ./
COPY web/static/css/ ./web/static/css/

# Install npm dependencies
RUN npm ci --only=production

# Build CSS and JS assets
RUN npm run build-assets

# Stage 2: Go build environment
FROM golang:1.24-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built assets from node-builder
COPY --from=node-builder /app/web/static/ ./web/static/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/xanthus .

# Stage 3: Production image
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 xanthus && \
    adduser -u 1000 -G xanthus -s /bin/sh -D xanthus

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/bin/xanthus .

# Copy static assets
COPY --from=go-builder /app/web/ ./web/

# Create data directory for persistence
RUN mkdir -p /app/data && chown -R xanthus:xanthus /app

# Switch to non-root user
USER xanthus

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ["/app/xanthus", "--health-check"] || exit 1

# Expose port
EXPOSE 8081

# Run the application
CMD ["/app/xanthus"]