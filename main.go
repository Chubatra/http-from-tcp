package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type Headers struct {
	fields map[string]string
}

func NewHeaders() *Headers {
	return &Headers{
		fields: make(map[string]string),
	}
}

func (h *Headers) Get(name string) string {
	return h.fields[strings.ToLower(name)]
}

func (h *Headers) Set(name, value string) {
	name = strings.ToLower(name)
	if existing, exists := h.fields[name]; exists {
		// Multiple headers with same name get comma-separated
		h.fields[name] = fmt.Sprintf("%s,%s", existing, value)
	} else {
		h.fields[name] = value
	}
}

func parseHeaderLine(line []byte) (string, string, error) {
	parts := bytes.SplitN(line, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed header line")
	}

	name := bytes.TrimSpace(parts[0])
	value := bytes.TrimSpace(parts[1])

	if len(name) == 0 {
		return "", "", fmt.Errorf("empty header name")
	}

	return string(name), string(value), nil
}

func (h *Headers) Parse(data []byte) (int, bool, error) {
	bytesRead := 0

	for {
		// Find next CRLF
		idx := bytes.Index(data[bytesRead:], []byte("\r\n"))
		if idx == -1 {
			break // Need more data
		}

		// Empty line means end of headers
		if idx == 0 {
			bytesRead += 2 // Skip the CRLF
			return bytesRead, true, nil
		}

		line := data[bytesRead : bytesRead+idx]
		name, value, err := parseHeaderLine(line)
		if err != nil {
			return 0, false, err
		}

		h.Set(name, value)
		bytesRead += idx + 2 // Skip line + CRLF
	}

	return bytesRead, false, nil // Headers not complete yet
}

type RequestLine struct {
	Method        string
	RequestTarget string
	HttpVersion   string
}

var (
	CRLF                    = []byte("\r\n")
	ErrMalformedRequestLine = fmt.Errorf("malformed request line")
)

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	// Find the end of the line
	idx := bytes.Index(data, CRLF)
	if idx == -1 {
		return nil, 0, nil // Need more data
	}

	line := data[:idx]
	bytesRead := idx + len(CRLF)

	// Split by spaces: "GET / HTTP/1.1" -> ["GET", "/", "HTTP/1.1"]
	parts := bytes.Split(line, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrMalformedRequestLine
	}

	// Validate HTTP version format
	httpParts := bytes.Split(parts[2], []byte("/"))
	if len(httpParts) != 2 ||
		string(httpParts[0]) != "HTTP" ||
		string(httpParts[1]) != "1.1" {
		return nil, 0, ErrMalformedRequestLine
	}

	return &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		HttpVersion:   string(httpParts[1]),
	}, bytesRead, nil
}

type Request struct {
	RequestLine RequestLine
	Headers     *Headers
	state       parserState
}

type parserState string

const (
	StateInit    parserState = "init"
	StateHeaders parserState = "headers"
	StateDone    parserState = "done"
	StateError   parserState = "error"
)

func NewRequest() *Request {
	return &Request{
		state:   StateInit,
		Headers: NewHeaders(),
	}
}

func (r *Request) parse(data []byte) (int, error) {
	totalRead := 0

	for {
		remaining := data[totalRead:]
		if len(remaining) == 0 {
			break
		}

		switch r.state {
		case StateInit:
			requestLine, bytesRead, err := parseRequestLine(remaining)
			if err != nil {
				r.state = StateError
				return 0, err
			}
			if bytesRead == 0 {
				break // Need more data
			}

			r.RequestLine = *requestLine
			totalRead += bytesRead
			r.state = StateHeaders

		case StateHeaders:
			bytesRead, done, err := r.Headers.Parse(remaining)
			if err != nil {
				r.state = StateError
				return 0, err
			}

			totalRead += bytesRead
			if done {
				r.state = StateDone
				return totalRead, nil
			}

		case StateDone, StateError:
			return totalRead, nil
		}
	}

	return totalRead, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := NewRequest()
	buffer := make([]byte, 1024)
	bufferLen := 0

	for request.state != StateDone && request.state != StateError {
		n, err := reader.Read(buffer[bufferLen:])
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read: %w", err)
		}

		bufferLen += n
		bytesRead, parseErr := request.parse(buffer[:bufferLen])

		if parseErr != nil {
			return nil, parseErr
		}

		// Shift remaining data to front of buffer
		copy(buffer, buffer[bytesRead:bufferLen])
		bufferLen -= bytesRead

		if err == io.EOF {
			break
		}
	}

	if request.state != StateDone {
		return nil, fmt.Errorf("incomplete request")
	}

	return request, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	req, err := RequestFromReader(conn)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		// Send 400 Bad Request
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// Simple response
	response := fmt.Sprintf("HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"Hello, World!")

	conn.Write([]byte(response))

	log.Printf("%s %s", req.RequestLine.Method, req.RequestLine.RequestTarget)
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("Server listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}
