package http_client

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http/httputil"
	urlpkg "net/url"
	"strconv"
	"strings"
)

// Options represent the options that a request will take
type Options struct {
	Method  string
	Headers map[string]string
	HTTPVer string
}

// Response represents a HTTP response which was returned
type Response struct {
	StatusCode        int
	StatusDescription string
	Headers           map[string]string
	Body              string
	HTTPVer           string
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

func Request(rawurl string, options *Options) (resp *Response, err error) {
	url, err := urlpkg.Parse(rawurl)
	if url.Host == "" {
		return &Response{}, fmt.Errorf("%q is not a valid URL\n", rawurl)
	}
	if err != nil {
		return &Response{}, err
	}

	host := url.Hostname()
	path := url.Path
	if path == "" {
		path = "/"
	}
	port := url.Port()
	if port == "" {
		port = "80"
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return &Response{}, err
	}
	defer conn.Close()

	if _, ok := options.Headers["Host"]; !ok {
		options.Headers["Host"] = host
	}

	fmt.Fprintf(conn, "%s %s %s\r\n", options.Method, path, options.HTTPVer)
	for k, v := range options.Headers {
		fmt.Fprintf(conn, "%s: %s\r\n", k, v)
	}
	fmt.Fprint(conn, "\r\n")

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

func ReadHeaders(reader *bufio.Reader) (headers map[string]string, err error) {
	headers = make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				return make(map[string]string), err
			}
			break
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}

		header := strings.Split(trimmed, ": ")
		headers[header[0]] = header[1]
	}

	return headers, err
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
	statusCode = int(statusCodeInt64)
	if err != nil {
		return "", 0, "", err
	}
	statusDescription = status[2]

	return httpVer, statusCode, statusDescription, err
}
