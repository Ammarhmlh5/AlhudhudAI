# Multistage Dockerfile for AL-HUDHUD Backend
# Optimized for performance and security

# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# Use -ldflags="-w -s" to reduce binary size
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o main .

# Stage 2: Final Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata libc6-compat

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/.env.example .env

# Expose API port
EXPOSE 8080

# Run the binary
CMD ["./main"]
