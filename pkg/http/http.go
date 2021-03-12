package http

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	urlpkg "net/url"
	"strings"
)

// Options represent the options that a request will take
type Options struct {
	Method  string
	Headers map[string]string
	HTTPVer string
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

	return NewResponse(conn)
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
