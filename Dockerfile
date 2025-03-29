# Build stage
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -o shorter ./cmd/app

# Final stage
FROM debian:bookworm-slim

# Install necessary dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 \
    bash \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/shorter .

# Create directory for database
RUN mkdir -p /app/data

# Create a startup script that can load from .env
RUN echo '#!/bin/bash\n\
# If .env file exists, load it\n\
if [ -f .env ]; then\n\
  export $(grep -v "^#" .env | xargs)\n\
fi\n\
\n\
# Run the application with environment variables\n\
exec ./shorter' > /app/start.sh && chmod +x /app/start.sh

# Set environment variables
ENV PORT=8080
ENV DATABASE_URL=/app/data/shorter.db

# Expose the port
EXPOSE 8080

# Run the startup script
CMD ["/app/start.sh"]
