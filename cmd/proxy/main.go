package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	urlpkg "net/url"
	"strings"

	"github.com/lexesjan/go-web-proxy-server/pkg/http_client"
)

func main() {
	port := 8080
	lc, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listning on \"http://localhost:%d\"\n", port)
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

func handleHttps(conn net.Conn, rawurl, httpVer string) {
	url, err := urlpkg.Parse(fmt.Sprintf("https://%s/", rawurl))
	if err != nil {
		log.Println(err)
		return
	}
	remote, err := net.Dial("tcp", url.Host)
	if err != nil {
		log.Println(err)
		return
	}
	defer remote.Close()

	fmt.Fprint(conn, "HTTP/1.1 200 Connection Established\r\n")
	fmt.Fprint(conn, "\r\n")
	go io.Copy(remote, conn)
	io.Copy(conn, remote)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	method, url, httpVer, err := readRequestStatus(reader)
	if err != nil {
		log.Println(err)
		return
	}

	if method == "CONNECT" {
		log.Printf("HTTPS Request %q %q\n", url, httpVer)
		handleHttps(conn, url, httpVer)
		return
	}

	requestHeaders, err := http_client.ReadHeaders(reader)
	if err != nil {
		log.Println(err)
		return
	}

	options := &http_client.Options{
		Method:  method,
		HTTPVer: httpVer,
		Headers: requestHeaders,
	}

	log.Printf("HTTP Request %q %q %q\n", options.Method, url, options.HTTPVer)
	resp, err := http_client.Request(url, options)
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
