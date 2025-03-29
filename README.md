# 🔗 Shorter - A URL Shortener Service

A simple, efficient URL shortener service built with Go and Chi router.

## Features

- ✂️ Create shortened URLs with optional custom short codes
- 🔄 Redirect from short URLs to original URLs
- 📱 Generate QR codes for short URLs
- 📊 View URL visit statistics
- 🔒 Protected API for creating short URLs (Basic Auth)
- ⚡ LRU caching for faster redirects
- 🧱 Domain-driven design architecture
- 📊 Comprehensive structured logging with slog

## Tech Stack

- Golang
- Chi HTTP Router
- SQLite
- Namespace-based LRU Cache
- QR Code Generation
- Structured Logging with slog

## API Endpoints

- `POST /api/urls` - Create a short URL (protected with Basic Auth)
- `GET /{shortCode}` - Redirect to the original URL
- `GET /api/urls/{shortCode}/stats` - Get URL statistics
- `GET /api/urls/{shortCode}/qrcode` - Generate a QR code for the short URL
- `GET /health` - Health check endpoint

## Installation & Setup

### Prerequisites

- Go 1.16 or later
- SQLite

### Local Development

1. Clone the repository
   ```
   git clone https://github.com/prasetyowira/shorter.git
   cd shorter
   ```

2. Install dependencies
   ```
   go mod download
   ```

3. Run the server
   ```
   go run cmd/app/main.go
   ```

The server will start at `http://localhost:8080` by default.

### Configuration

The application can be configured using environment variables:

| Variable     | Description                     | Default            |
|--------------|--------------------------------|-------------------|
| PORT         | HTTP server port               | 8080              |
| DATABASE_URL | SQLite database path           | shorter.db        |
| AUTH_USER    | Basic Auth username            | admin             |
| AUTH_PASS    | Basic Auth password            | password          |
| BASE_URL     | Base URL for short URLs        | http://localhost:8080 |
| CACHE_SIZE   | Size of the LRU cache          | 1000              |
| LOG_LEVEL    | Logging level (DEBUG, INFO, WARN, ERROR) | INFO              |

#### Using .env File

You can also use a `.env` file to configure the application instead of setting environment variables:

```
PORT=8080
DATABASE_URL=shorter.db
AUTH_USER=admin
AUTH_PASS=password
BASE_URL=http://localhost:8080
CACHE_SIZE=1000
LOG_LEVEL=INFO
```

When running with Docker, the container is configured to automatically load variables from a `.env` file if it's mounted in the container.

## Usage Examples

### Create a Short URL

```bash
curl -X POST http://localhost:8080/api/urls \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"long_url": "https://example.com/very/long/url"}'
```

Response:
```json
{
  "short_code": "abc123",
  "long_url": "https://example.com/very/long/url"
}
```

### Create a Short URL with Custom Code

```bash
curl -X POST http://localhost:8080/api/urls \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"long_url": "https://example.com/very/long/url", "custom_short_url": "custom"}'
```

Response:
```json
{
  "short_code": "custom",
  "long_url": "https://example.com/very/long/url"
}
```

### Get URL Statistics

```bash
curl -X GET http://localhost:8080/api/urls/abc123/stats
```

Response:
```json
{
  "short_code": "abc123",
  "visits": 42
}
```

### Get QR Code

Access the QR code in your browser:
```
http://localhost:8080/api/urls/abc123/qrcode
```

Or using curl:
```bash
curl -X GET http://localhost:8080/api/urls/abc123/qrcode --output qrcode.png
```

This returns a PNG image of a QR code that, when scanned, redirects to the original URL.

## Logging

The application uses structured logging with slog, providing:

- Request/response logging with unique request IDs
- Detailed error information including error codes and types
- Performance metrics for database operations
- Different log levels (DEBUG, INFO, WARN, ERROR)
- Request tracing via X-Request-ID header

Each log entry includes:
- Timestamp
- Log level
- Request ID
- Context/function name
- File name
- Message
- Additional data fields

Set the `LOG_LEVEL` environment variable to control logging verbosity:
- `DEBUG`: Detailed development information
- `INFO`: Important operational events (default)
- `WARN`: Unexpected but handled conditions
- `ERROR`: Critical issues requiring immediate attention

## Deployment

### Using Docker

1. Build the Docker image
   ```
   docker build -t shorter:latest .
   ```

2. Run the container with environment variables
   ```
   docker run -p 8080:8080 -e BASE_URL=https://yourdomain.com shorter:latest
   ```

3. Run with .env file
   ```
   # Create a .env file with your configuration
   echo "PORT=8080\nBASE_URL=https://yourdomain.com\nAUTH_USER=admin\nAUTH_PASS=securepassword" > .env
   
   # Mount the .env file when running the container
   docker run -p 8080:8080 -v $(pwd)/.env:/app/.env shorter:latest
   ```

### Deploying to Railway.app

1. Create a new project on Railway.app
2. Connect your GitHub repository
3. Set the required environment variables in Railway dashboard
4. Railway will automatically build and deploy your application

## License

MIT