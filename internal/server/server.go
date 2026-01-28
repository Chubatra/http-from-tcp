package server

import (
	"fmt"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"io"
	"net"
)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
	Port    uint16
	closed  bool
	Handler Handler
}

func (s *Server) runConnection(conn io.ReadWriteCloser) {
	defer conn.Close()
	responseWriter := response.NewWriter(conn)
	r, err := request.RequestFromReader(conn)
	if err != nil {
		responseWriter.WriteStatusLine(response.StatusBadRequest)
		responseWriter.WriteHeaders(*response.GetDefaultHeaders(0))
		return
	}
	s.Handler(responseWriter, r)
}

func (s *Server) runServer(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if s.closed {
			return
		}
		if err != nil {
			return
		}
		go s.runConnection(conn)
	}
}

func Serve(port uint16, handler Handler) (*Server, error) {
	s := &Server{
		Port:    port,
		closed:  false,
		Handler: handler,
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return nil, err
	}
	go s.runServer(listener)
	return s, nil
}

func (s *Server) Close() error {
	s.closed = true
	return nil
}
