package main

import (
	"fmt"
	"io"
	"log"
	"net"
	urlpkg "net/url"

	"github.com/lexesjan/go-web-proxy-server/pkg/http"
	"github.com/lexesjan/go-web-proxy-server/pkg/httpclient"
)

func main() {
	port := 8080
	lc, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[ Listning on \"http://localhost:%d\" ]\n", port)
	defer lc.Close()

	for {
		conn, err := lc.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(conn)
	}
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

	req, err := http.NewRequest(conn)
	if err != nil {
		log.Println(err)
	}

	host := req.Headers["Host"]
	if req.Method == "CONNECT" {
		log.Printf("[ HTTPS Request %q %q ]\n", host, req.HTTPVer)
		handleHttps(conn, host, req.HTTPVer)
		return
	}
	reqUrl := fmt.Sprintf("http://%s", host)
	log.Printf("[ HTTP Request %q %q %q ]\n", req.Method, reqUrl, req.HTTPVer)
	resp, err := httpclient.Request(reqUrl,
		&httpclient.Options{
			Method:  req.Method,
			HTTPVer: req.HTTPVer,
			Headers: req.Headers,
		},
	)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Fprint(conn, resp)
}
