package request

import (
	"bytes"
	"fmt"
	"httpfromtcp/internal/headers"
	"io"
)

type parserState string

const (
	StateInit    parserState = "init"
	StateDone    parserState = "done"
	StateError   parserState = "error"
	StateHeaders parserState = "headers"
	StateBody    parserState = "body"
)

type RequestLine struct {
	Method        string
	RequestTarget string
	HttpVersion   string
}

type Request struct {
	RequestLine RequestLine
	Headers     *headers.Headers
	Body        string
	state       parserState
}

func newRequest() *Request {
	return &Request{
		state:       StateInit,
		RequestLine: RequestLine{},
		Headers:     headers.NewHeaders(),
		Body:        "",
	}
}

var (
	ErrMalformedRequestLine   = fmt.Errorf("malformed request line")
	ErrUnsupportedHttpVersion = fmt.Errorf("unsupported http version")
	ErrRequestInErrState      = fmt.Errorf("request in error state")
	ErrIncompleteBody         = fmt.Errorf("incomplete body: received less data than content-length")
	SAPERATOR                 = []byte("\r\n")
)

func parseRequestLine(b []byte) (*RequestLine, int, error) {
	idx := bytes.Index(b, SAPERATOR)
	if idx == -1 {
		return nil, 0, nil
	}

	startLine := b[:idx]
	read := idx + len(SAPERATOR)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrMalformedRequestLine
	}

	httpParts := bytes.Split(parts[2], []byte("/"))
	if len(httpParts) != 2 || string(httpParts[0]) != "HTTP" || string(httpParts[1]) != "1.1" {
		return nil, 0, ErrMalformedRequestLine
	}

	rl := &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		HttpVersion:   string(httpParts[1]),
	}
	return rl, read, nil
}

func (r *Request) hasBody() bool {
	//TODO: fix when chunck encoding
	length := r.Headers.GetInt("content-length", 0)
	return length > 0
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
dance:
	for {
		currentData := data[read:]
		if len(currentData) == 0 {
			break dance
		}
		switch r.state {

		case StateError:
			return 0, ErrRequestInErrState
		case StateInit:
			rl, n, err := parseRequestLine(currentData)
			if err != nil {
				r.state = StateError
				return 0, err
			}
			if n == 0 {
				break dance
			}
			r.RequestLine = *rl
			read += n
			r.state = StateHeaders
		case StateDone:
			break dance

		case StateHeaders:
			n, done, err := r.Headers.Parse(currentData)
			if err != nil {
				r.state = StateError
				return 0, err
			}
			if done {
				if r.hasBody() {
					r.state = StateBody
				} else {
					r.state = StateDone
				}
			}
			read += n
			if n == 0 {
				break dance
			}
		case StateBody:
			length := r.Headers.GetInt("content-length", 0)
			if length == 0 {
				panic("chunck encoding not implemented")
			}

			remaining := min(length-len(r.Body), len(currentData))
			r.Body += string(currentData[:remaining])
			read += remaining

			if len(r.Body) == length {
				r.state = StateDone
			}

		default:
			panic("invalid state")
		}
	}
	return read, nil

}

func (r *Request) done() bool {
	return r.state == StateDone || r.state == StateError
}
func RequestFromReader(reader io.Reader) (*Request, error) {

	request := newRequest()

	// NOTE: buffer can be longer than 1024
	buf := make([]byte, 1024)
	bufLen := 0

	for !request.done() {
		n, err := reader.Read(buf[bufLen:])
		// TODO: what to do here?
		if err != nil {
			break
		}

		bufLen += n
		readN, err := request.parse(buf[:bufLen])

		if err != nil {
			return nil, err
		}

		copy(buf, buf[readN:bufLen])
		bufLen -= readN
	}
	if request.state == StateBody {
		return nil, ErrIncompleteBody
	}
	return request, nil
}
