package http

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	urlpkg "net/url"
	"strconv"
	"strings"
)

// Request represents a HTTP request.
type Request struct {
	Method  string
	Path    string
	HTTPVer string
	Headers map[string]string
	Body    string
}

// NewRequest returns a new Request created by reading the connection and
// parsing the HTTP request message.
func NewRequest(conn net.Conn) (req *Request, err error) {
	reader := bufio.NewReader(conn)
	method, path, httpVer, err := readRequestStatus(reader)
	if err != nil {
		return &Request{}, err
	}

	// Proxy HTTP request.
	if !strings.HasPrefix(path, "/") {
		url, err := urlpkg.Parse(fmt.Sprintf("http://%s", path))
		if err != nil {
			return &Request{}, err
		}

		path = url.Path
	}

	requestHeaders, err := ReadHeaders(reader)
	if err != nil {
		return &Request{}, err
	}

	req = &Request{
		Method:  method,
		Path:    path,
		HTTPVer: httpVer,
		Headers: requestHeaders,
		Body:    "",
	}

	// Parse body if exists.
	if _, ok := requestHeaders["Content-Length"]; ok {
		contentLength, err := strconv.ParseInt(requestHeaders["Content-Length"], 10, 0)
		if err != nil {
			return &Request{}, err
		}

		body, err := ioutil.ReadAll(io.LimitReader(reader, contentLength))
		if err != nil {
			return &Request{}, err
		}

		req.Body = string(body)
	}

	return req, nil
}

func readRequestStatus(reader *bufio.Reader) (method, url, httpVer string, err error) {
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return "", "", "", err
	}

	trimmed := strings.TrimRight(statusLine, "\r\n")
	status := strings.Split(trimmed, " ")
	method = status[0]
	url = status[1]
	httpVer = status[2]

	return method, url, httpVer, nil
}

func (req *Request) String() (str string) {
	var builder strings.Builder

	fmt.Fprintf(&builder, "%s %s %s\r\n", req.Method, req.Path, req.HTTPVer)
	for k, v := range req.Headers {
		fmt.Fprintf(&builder, "%s: %s\r\n", k, v)
	}
	fmt.Fprint(&builder, "\r\n")
	fmt.Fprint(&builder, req.Body)

	return builder.String()
}
