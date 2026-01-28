package main

import (
	"crypto/sha256"
	"fmt"
	"httpfromtcp/internal/headers"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"httpfromtcp/internal/server"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const port = 42069

func toStr(b []byte) string {
	out := ""
	for _, bb := range b {
		out += fmt.Sprintf("%02x", bb)
	}
	return out
}

func main() {
	s, err := server.Serve(port, func(w *response.Writer, req *request.Request) {
		h := response.GetDefaultHeaders(0)

		text := "<html> <head> <title>%s</title> </head> <body> <h1>%s</h1> <p>%s</p> </body> </html>"

		status := 200
		body := fmt.Sprintf(text, "200 OK", "Success!", "Your request was an absolute banger.")

		if req.RequestLine.RequestTarget == "/yourproblem" {
			status = 400
			body = fmt.Sprintf(text, "400 Bad Request", "Bad Request", "Your request honestly kinda sucked.")

		} else if req.RequestLine.RequestTarget == "/myproblem" {
			status = 500
			body = fmt.Sprintf(text, "500 Internal Server Error", "Internal Server Error", "Okay, you know what? This one is on me.")

		} else if req.RequestLine.RequestTarget == "/video" {
			f, err := os.ReadFile("assets/vim.mp4")
			if err == nil {
				h.Replace("Content-type", "video/mp4")
				h.Replace("Content-length", fmt.Sprintf("%d", len(f)))

				w.WriteStatusLine(response.StatusOK)
				w.WriteHeaders(*h)
				w.WriteBody(f)
				return
			}

			status = 500
			body = fmt.Sprintf(text, "500 Internal Server Error", fmt.Sprintf("%v", err), "Okay, you know what? This one is on me.")

		} else if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin/stream") {

			target := req.RequestLine.RequestTarget
			res, err := http.Get("https://httpbin.org/" + target[len("/httpbin/"):])

			if err == nil {
				w.WriteStatusLine(response.StatusOK)

				h.Delete("Content-length")
				h.Set("Transfer-Encoding", "chunked")
				h.Replace("Content-type", "text/plain")

				h.Set("Trailer", "X-Content-SHA256")
				h.Set("Trailer", "X-Content-Length")

				w.WriteHeaders(*h)

				fullBody := []byte{}

				for {
					data := make([]byte, 32)
					n, err := res.Body.Read(data)
					if err != nil {
						break
					}
					fullBody = append(fullBody, data[:n]...)

					w.WriteBody(fmt.Appendf(nil, "%x\r\n", n))
					w.WriteBody(data[:n])
					w.WriteBody([]byte("\r\n"))

				}
				w.WriteBody([]byte("0\r\n"))

				trailer := headers.NewHeaders()
				hash := sha256.Sum256(fullBody)
				trailer.Set("X-Content-SHA256", toStr(hash[:]))
				trailer.Set("X-Content-Length", fmt.Sprintf("%d", len(fullBody)))
				w.WriteHeaders(*trailer)
				return
			}

			status = 500
			body = fmt.Sprintf(text, "500 Internal Server Error", "Internal Server Error", "Okay, you know what? This one is on me.")
		}

		h.Replace("Content-type", "text/html")
		h.Replace("Content-length", fmt.Sprintf("%d", len(body)))

		w.WriteStatusLine(response.StatusCode(status))
		w.WriteHeaders(*h)
		w.WriteBody([]byte(body))
	})

	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer s.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
