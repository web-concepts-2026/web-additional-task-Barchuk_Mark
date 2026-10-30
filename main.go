package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Headers map[string]string
	Body    []byte
}

func main() {

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("Server listening on http://localhost:8080")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	request, err := parseRequest(bufio.NewReader(conn))
	if err != nil {
		writeResponse(conn, 400, "text/plain; charset=utf-8", []byte("Bad Request\n"))
		return
	}

	log.Printf("%s %s", request.Method, request.Path)

	switch {
	case request.Method == "GET" && (request.Path == "/" || request.Path == "/page"):
		page, err := os.ReadFile("static/index.html")
		if err != nil {
			writeResponse(conn, 500, "text/plain; charset=utf-8", []byte("Cannot read HTML file\n"))
			return
		}
		message := "No page ID provided"
		if id := request.Query.Get("id"); id != "" {
			message = "Page ID: " + html.EscapeString(id)
		}
		page = []byte(strings.ReplaceAll(string(page), "{{PAGE_MESSAGE}}", message))
		writeResponse(conn, 200, "text/html; charset=utf-8", page)

	case request.Method == "POST" && request.Path == "/echo":
		contentType := strings.ToLower(request.Headers["content-type"])
		if !strings.HasPrefix(contentType, "application/json") {
			writeJSON(conn, 415, map[string]string{"error": "Content-Type must be application/json"})
			return
		}

		var input json.RawMessage
		if err := json.Unmarshal(request.Body, &input); err != nil || input == nil {
			writeJSON(conn, 400, map[string]string{"error": "Expected a JSON object"})
			return
		}

		writeResponse(conn, 200, "application/json; charset=utf-8", input)

	default:
		writeResponse(conn, 404, "text/plain; charset=utf-8", []byte("Not Found\n"))
	}
}

func parseRequest(reader *bufio.Reader) (*Request, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	parts := strings.Fields(line)
	if len(parts) != 3 || parts[2] != "HTTP/1.1" {
		return nil, fmt.Errorf("invalid request line")
	}

	u, err := url.ParseRequestURI(parts[1])
	if err != nil {
		return nil, err
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}

	request := &Request{
		Method:  parts[0],
		Path:    u.Path,
		Query:   query,
		Headers: make(map[string]string),
	}

	totalHeaderBytes := 0
	for i := 0; i < 100; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		totalHeaderBytes += len(line)
		if totalHeaderBytes > 16*1024 {
			return nil, fmt.Errorf("headers too large")
		}
		if line == "\r\n" {
			break
		}
		key, value, found := strings.Cut(strings.TrimRight(line, "\r\n"), ":")
		if !found || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid header")
		}
		request.Headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
		if i == 99 {
			return nil, fmt.Errorf("too many headers")
		}
	}

	if value, ok := request.Headers["content-length"]; ok {
		size, err := strconv.Atoi(value)
		if err != nil || size < 0 || size > 1024*1024 {
			return nil, fmt.Errorf("invalid content length")
		}
		request.Body = make([]byte, size)
		if _, err := io.ReadFull(reader, request.Body); err != nil {
			return nil, err
		}
	}
	return request, nil
}

func writeJSON(conn net.Conn, status int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		writeResponse(conn, 500, "text/plain", []byte("Internal Server Error\n"))
		return
	}
	writeResponse(conn, status, "application/json; charset=utf-8", body)
}

func writeResponse(conn net.Conn, status int, contentType string, body []byte) {
	statusText := map[int]string{200: "OK", 400: "Bad Request", 404: "Not Found", 415: "Unsupported Media Type", 500: "Internal Server Error"}[status]
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\nConnection: close\r\n\r\n", status, statusText, contentType, len(body))
	io.WriteString(conn, response)
	conn.Write(body)
}
