package main

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

func main() {
	port := 8080
	lc, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Listning on http://localhost:%d\n", port)
	defer lc.Close()

	for {
		conn, err := lc.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(conn)
	}
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

func readHostHeader(reader *bufio.Reader) (hostURL string, port string, err error) {
	hostHeaderLine, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	trimmed := strings.TrimRight(hostHeaderLine, "\r\n")
	hostHeader := strings.Split(trimmed, ": ")
	hostPortPair := strings.Split(hostHeader[1], ":")

	if len(hostPortPair) == 1 {
		return hostPortPair[0], "80", nil
	}

	return hostPortPair[0], hostPortPair[1], nil
}

// Options struct
type Options struct {
	Method  string
	Headers map[string]string
	HTTPVer string
}

// Response struct
type Response struct {
	StatusCode        int
	StatusDescription string
	Headers           map[string]string
	Body              string
	HTTPVer           string
}

func request(rawurl string, options *Options) (resp *Response, err error) {
	url, err := urlpkg.Parse(rawurl)
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

	responseHeaders := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return &Response{}, err
			}
			break
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}

		header := strings.Split(trimmed, ": ")
		responseHeaders[header[0]] = header[1]
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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	method, url, httpVer, err := readRequestStatus(reader)

	if method == "CONNECT" {
		return
	}

	requestHeaders := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				return
			}
			break
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}

		header := strings.Split(trimmed, ": ")
		requestHeaders[header[0]] = header[1]
	}

	options := &Options{
		Method:  method,
		HTTPVer: httpVer,
		Headers: requestHeaders,
	}

	resp, err := request(url, options)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Fprintf(conn, "%s %d %s\r\n", resp.HTTPVer, resp.StatusCode, resp.StatusDescription)
	for k, v := range resp.Headers {
		fmt.Fprintf(conn, "%s: %s\r\n", k, v)
	}
	fmt.Fprint(conn, "\r\n")
	fmt.Fprint(conn, resp.Body)
}
