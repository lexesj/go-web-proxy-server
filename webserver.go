package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
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

func readRequestStatus(reader *bufio.Reader) (string, string, string, error) {
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return "", "", "", err
	}
	trimmed := strings.TrimRight(statusLine, "\r\n")

	status := strings.Split(trimmed, " ")
	method := status[0]
	url := status[1]
	httpVer := status[2]

	return method, url, httpVer, nil
}

func readResponseStatus(reader *bufio.Reader) (string, int, string, error) {
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return "", 0, "", err
	}
	trimmed := strings.TrimRight(statusLine, "\r\n")

	status := strings.Split(trimmed, " ")
	httpVer := status[0]
	statusCode, err := strconv.ParseInt(status[1], 10, 0)
	if err != nil {
		return "", 0, "", err
	}
	statusDescription := status[2]

	return httpVer, int(statusCode), statusDescription, err
}

func readHostHeader(reader *bufio.Reader) (string, string, error) {
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

func request(rawurl string, options *Options) (*Response, error) {
	url, err := urlpkg.Parse(rawurl)
	if err != nil {
		return &Response{}, err
	}

	port := url.Port()
	host := url.Hostname()
	if port == "" {
		port = "80"
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return &Response{}, err
	}
	defer conn.Close()

	if _, ok := options.Headers["Host"]; !ok {
		options.Headers["Host"] = url.Hostname()
	}

	var requestHeaders strings.Builder
	for k, v := range options.Headers {
		fmt.Fprintf(&requestHeaders, "%s: %s\r\n", k, v)
	}

	fmt.Fprintf(conn, "%s %s %s\r\n", options.Method, url.Path, options.HTTPVer)
	fmt.Fprintf(conn, "%s\r\n", requestHeaders.String())
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

	contentLength, err := strconv.ParseInt(responseHeaders["Content-Length"], 10, 0)
	if err != nil {
		return &Response{}, err
	}

	body, err := ioutil.ReadAll(io.LimitReader(reader, contentLength))
	if err != nil {
		return &Response{}, err
	}

	resp := &Response{
		StatusCode:        statusCode,
		StatusDescription: statusDescription,
		Headers:           responseHeaders,
		Body:              string(body),
		HTTPVer:           httpVer,
	}

	return resp, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	options := &Options{
		Method:  "GET",
		HTTPVer: "HTTP/1.1",
		Headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64; rv:86.0) Gecko/20100101 Firefox/86.0",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.5",
			"Accept-Encoding": "gzip, deflate",
			"Connection":      "keep-alive",
		},
	}
	resp, err := request("http://www.google.com/", options)
	if err != nil {
		log.Print(err)
	}

	fmt.Println(resp)
}
