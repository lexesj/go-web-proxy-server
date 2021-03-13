package http

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http/httputil"
	"strconv"
	"strings"
)

// Response represents a HTTP response.
type Response struct {
	StatusCode        int
	StatusDescription string
	Headers           map[string]string
	Body              string
	HTTPVer           string
}

// NewResponse returns a new Response created by reading the connection and
// parsing the HTTP response message.
func NewResponse(conn net.Conn) (resp *Response, err error) {
	reader := bufio.NewReader(conn)
	httpVer, statusCode, statusDescription, err := readResponseStatus(reader)
	if err != nil {
		return &Response{}, err
	}

	responseHeaders, err := ReadHeaders(reader)
	if err != nil {
		return &Response{}, err
	}

	resp = &Response{
		StatusCode:        statusCode,
		StatusDescription: statusDescription,
		Headers:           responseHeaders,
		Body:              "",
		HTTPVer:           httpVer,
	}

	// Parse Body if exists
	if _, ok := responseHeaders["Content-Length"]; ok {
		contentLength, err := strconv.ParseInt(responseHeaders["Content-Length"], 10, 0)
		if err != nil {
			return &Response{}, err
		}
		body, err := ioutil.ReadAll(io.LimitReader(reader, contentLength))
		if err != nil {
			return &Response{}, err
		}
		resp.Body = string(body)
	} else if responseHeaders["Transfer-Encoding"] == "chunked" {
		delete(responseHeaders, "Transfer-Encoding")
		body, err := ioutil.ReadAll(httputil.NewChunkedReader(reader))
		if err != nil {
			return &Response{}, err
		}
		resp.Body = string(body)
	}

	return resp, nil
}

func readResponseStatus(reader *bufio.Reader) (httpVer string, statusCode int, statusDescription string, err error) {
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return "", 0, "", err
	}

	trimmed := strings.TrimRight(statusLine, "\r\n")
	status := strings.Split(trimmed, " ")
	httpVer = status[0]
	statusCodeInt64, err := strconv.ParseInt(status[1], 10, 0)
	if err != nil {
		return "", 0, "", err
	}

	statusCode = int(statusCodeInt64)
	statusDescription = status[2]

	return httpVer, statusCode, statusDescription, err
}

func (resp *Response) String() (str string) {
	var builder strings.Builder

	fmt.Fprintf(&builder, "%s %d %s\r\n", resp.HTTPVer, resp.StatusCode, resp.StatusDescription)
	for k, v := range resp.Headers {
		fmt.Fprintf(&builder, "%s: %s\r\n", k, v)
	}
	fmt.Fprint(&builder, "\r\n")
	fmt.Fprint(&builder, resp.Body)

	return builder.String()
}
