package headers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParse(t *testing.T) {
	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers.Get("Host"))
	assert.Equal(t, 25, n)
	assert.True(t, done)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("       Host : localhost:42069       \r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("H©st: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("Host: localhost:42069\r\nSet-Person: lane-loves-go\r\nSet-Person: prime-loves-zig\r\nSet-Person: tj-loves-ocaml\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "lane-loves-go,prime-loves-zig,tj-loves-ocaml", headers.Get("Set-Person"))
	assert.Equal(t, 109, n)
	assert.True(t, done)
}

func TestHeaderParseWithCurlPost(t *testing.T) {
	data := []byte(
		"POST /coffee HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"User-Agent: curl/8.15.0\r\n" +
			"Accept: */*\r\n" +
			"Content-Type: application/json\r\n" +
			"Content-Length: 39\r\n" +
			"\r\n" +
			"{\"type\": \"dark mode\", \"size\": \"medium\"}",
	)

	// Only parse the headers portion (stop at \r\n\r\n)
	headers := NewHeaders()
	n, done, err := headers.Parse(data[len("POST /coffee HTTP/1.1\r\n"):]) // skip request line
	require.NoError(t, err)
	require.True(t, done)
	require.NotNil(t, headers)

	// Assertions
	assert.Equal(t, "localhost:42069", headers.Get("Host"))
	assert.Equal(t, "curl/8.15.0", headers.Get("User-Agent"))
	assert.Equal(t, "*/*", headers.Get("Accept"))
	assert.Equal(t, "application/json", headers.Get("Content-Type"))
	assert.Equal(t, "39", headers.Get("Content-Length"))

	// Ensure parser consumed all headers (right before body starts)
	assert.Equal(t, strings.Index(string(data), "\r\n\r\n")+4-len("POST /coffee HTTP/1.1\r\n"), n)
}
