FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -o shorter ./cmd/server

# Use a smaller image for the final build
FROM alpine:latest

# SQLite dependencies
RUN apk add --no-cache libc6-compat

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/shorter .

# Create directory for database
RUN mkdir -p /app/data

# Set environment variables
ENV PORT=8080
ENV DATABASE_URL=/app/data/shorter.db
ENV GIN_MODE=release

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./shorter"] 