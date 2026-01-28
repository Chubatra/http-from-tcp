# HTTP from TCP

A lightweight HTTP/1.1 server implementation built from scratch in Go, parsing TCP connections directly without using the standard `net/http` package.

## Features

- **HTTP/1.1 Protocol Support**: Handles request parsing, headers, and bodies
- **Chunked Transfer Encoding**: Supports streaming responses with trailers
- **Custom Routes**: Includes demo endpoints with different status codes
- **Video Streaming**: Serves video files with proper content types
- **HTTP Proxying**: Can proxy requests to external services (e.g., httpbin.org)

## Getting Started

### Prerequisites

- Go 1.16 or higher
- Git (for cloning the repository)

### Installation

```bash
# Clone the repository
git clone https://github.com/shv-ng/http-from-tcp
cd http-from-tcp

# Install dependencies
go mod download
```

### Running the Server

```bash
# Start the HTTP server
go run cmd/httpserver/main.go
```

The server will start on port **42069**.

### Running the TCP Listener (Debug Tool)

For debugging raw HTTP requests:

```bash
go run cmd/tcplistener/main.go
```

## Usage

### Basic Requests

Once the server is running, you can make requests:

```bash
# Basic GET request
curl http://localhost:42069/

# Test error responses
curl http://localhost:42069/yourproblem  # Returns 400
curl http://localhost:42069/myproblem    # Returns 500

# Stream video content
curl http://localhost:42069/video

# Test chunked transfer encoding with trailers
curl http://localhost:42069/httpbin/stream/20 -v
```

### Available Endpoints

- **`/`** - Returns a 200 OK response with HTML
- **`/yourproblem`** - Returns a 400 Bad Request
- **`/myproblem`** - Returns a 500 Internal Server Error
- **`/video`** - Streams a video file (requires `assets/vim.mp4`)
- **`/httpbin/stream/*`** - Proxies streaming requests to httpbin.org with chunked encoding

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/request
go test ./internal/headers
```

## Development

The project is organized into several internal packages:

- **request** - HTTP request parsing
- **response** - HTTP response writing
- **headers** - Header parsing and manipulation
- **server** - TCP server and connection handling

### Making Changes

1. Make your changes to the codebase
2. Run tests to ensure nothing breaks
3. Test manually with curl or your browser

### Stopping the Server

Press `Ctrl+C` to gracefully stop the server.

## Examples

### POST Request with Body

```bash
curl -X POST http://localhost:42069/ \
  -H "Content-Type: application/json" \
  -d '{"type": "dark mode", "size": "medium"}'
```

### Viewing Response Headers

```bash
curl -i http://localhost:42069/
```

### Testing Chunked Encoding

```bash
curl http://localhost:42069/httpbin/stream/10 -v
```

This will show chunked response with SHA256 hash and content length in trailers.

## License

MIT License - see [LICENSE](LICENSE) file for details
