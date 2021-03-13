package main

import (
	"fmt"
	"io"
	"log"
	"net"
	urlpkg "net/url"
	"strconv"
	"sync"
	"time"

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

	var cache sync.Map
	var blockList sync.Map

	for {
		conn, err := lc.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(conn, &cache, &blockList)
	}
}

func handleHTTPS(conn net.Conn, rawurl, httpVer string) (err error) {
	url, err := urlpkg.Parse(fmt.Sprintf("https://%s/", rawurl))
	if err != nil {
		return err
	}
	remote, err := net.Dial("tcp", url.Host)
	if err != nil {
		return err
	}
	defer remote.Close()

	fmt.Fprint(conn, "HTTP/1.1 200 Connection Established\r\n")
	fmt.Fprint(conn, "\r\n")

	// Tunnel between client and server.
	go io.Copy(remote, conn)
	io.Copy(conn, remote)

	return nil
}

func handleConnection(conn net.Conn, cache, blockList *sync.Map) {
	defer conn.Close()

	req, err := http.NewRequest(conn)
	if err != nil {
		log.Printf("[ Error %q ]\n", err)
		return
	}

	host := req.Headers["Host"]
	// Handle website blocking.
	if blocked, ok := blockList.Load(host); ok {
		if blocked == true {
			log.Printf("[ Blocked %q ]\n", host)
			forbiddenMessage := fmt.Sprintf("Blocked %q by proxy\n", host)
			respHeaders := map[string]string{"Content-Length": strconv.Itoa(len(forbiddenMessage))}
			resp := &http.Response{
				StatusCode:        403,
				StatusDescription: "Forbidden",
				Headers:           respHeaders,
				Body:              forbiddenMessage,
				HTTPVer:           req.HTTPVer,
			}
			fmt.Fprint(conn, resp)
			return
		}
	}

	// Handle HTTPS request.
	if req.Method == "CONNECT" {
		log.Printf("[ HTTPS %q %q %q ]\n", req.Method, host, req.HTTPVer)
		err := handleHTTPS(conn, host, req.HTTPVer)
		if err != nil {
			log.Printf("[ Error %q ]\n", err)
		}
		return
	}

	// Handle HTTP request.
	cachedResponse, cacheFound := cache.Load(host)
	if cacheFound {
		currTimeFormatted := time.Now().In(time.UTC).Format(http.TimeFormat)
		req.Headers["If-Modified-Since"] = currTimeFormatted
	}
	reqURL := fmt.Sprintf("http://%s", host)
	resp, err := httpclient.Request(reqURL,
		&httpclient.Options{
			Method:  req.Method,
			HTTPVer: req.HTTPVer,
			Headers: req.Headers,
		},
	)
	if err != nil {
		log.Printf("[ Error %q ]\n", err)
		return
	}
	// Cache is still valid. Return cached response.
	if cacheFound && resp.StatusCode == 304 {
		log.Printf("[ HTTP Response Cached %q %q %q ]\n", req.Method, reqURL, req.HTTPVer)
		fmt.Fprint(conn, cachedResponse)
		return
	}

	// Forward response to client.
	log.Printf("[ HTTP Response %q %q %q ]\n", req.Method, reqURL, req.HTTPVer)
	fmt.Fprint(conn, resp)
	cache.Store(host, resp.String())
}
