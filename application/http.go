package application

import (
	"bufio"
	"fmt"
	"strings"
	"unicode/utf8"
)

type HTTPRequest struct {
	Method  string
	URI     string
	Version string
	Headers map[string]string
	Body    string
}

type HTTPResponse struct {
	Version string
	Status  string
	Headers map[string]string
	Body    string
}

const (
	HTTP_VERSION = "HTTP/1.0"
)

// Parse an HTTP request from a string.
func ParseHTTPRequest(raw string) (*HTTPRequest, error) {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var requestLine string

	if scanner.Scan() {
		requestLine = scanner.Text()
	} else {
		return nil, fmt.Errorf("failed to read request line")
	}

	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}

	request := &HTTPRequest{
		Method:  parts[0],
		URI:     parts[1],
		Version: parts[2],
		Headers: make(map[string]string),
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}

		headerParts := strings.SplitN(line, ": ", 2)
		if len(headerParts) != 2 {
			return nil, fmt.Errorf("invalid header: %s", line)
		}

		request.Headers[headerParts[0]] = headerParts[1]
	}

	if request.Method == "POST" || request.Method == "PUT" {
		for scanner.Scan() {
			request.Body += scanner.Text()
		}
	}

	return request, nil
}

// Create a new HTTP response.
func NewTextOkResponse(body string) *HTTPResponse {
	res := HTTPResponse{
		Version: HTTP_VERSION,
		Status:  "200 OK",
		Headers: map[string]string{
			"Content-Type":   "text/plain",
			"Content-Length": fmt.Sprintf("%d", getContentLength(body)),
		},
		Body: body,
	}

	return &res
}

func getContentLength(body string) int {
	return utf8.RuneCountInString(body)
}

// Convert an HTTP response to a string.
func (res *HTTPResponse) String() string {
	var response string

	response += fmt.Sprintf("%s %s\r\n", res.Version, res.Status)

	for key, value := range res.Headers {
		response += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	response += "\r\n"

	response += res.Body

	return response
}
