# Multi-stage Dockerfile for SpellingClash
FROM golang:1.25-alpine AS builder

# Install build dependencies for both architectures
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set build arguments
ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /spellingclash ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

# Copy binary from builder
COPY --from=builder /spellingclash /app/spellingclash

# Copy templates, static files, migrations, and data
COPY internal/templates /app/internal/templates
COPY static /app/static
COPY migrations /app/migrations
COPY data /app/data

# Create directory for database and audio files
RUN mkdir -p /app/db /app/static/audio

# Set environment variables
ENV PORT=8080
ENV DB_PATH=/app/db/spellingclash.db
ENV AUDIO_DIR=/app/static/audio

# Expose port
EXPOSE 8080

# Create volume for persistent data
VOLUME ["/app/db", "/app/static/audio"]

# Run the application
CMD ["/app/spellingclash"]
