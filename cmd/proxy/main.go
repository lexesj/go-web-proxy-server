package main

import (
	"fmt"
	"io"
	logpkg "log"
	"net"
	urlpkg "net/url"
	"strconv"
	"sync"
	"time"

	"github.com/lexesjan/go-web-proxy-server/pkg/commandline"
	"github.com/lexesjan/go-web-proxy-server/pkg/http"
	"github.com/lexesjan/go-web-proxy-server/pkg/httpclient"
	"github.com/lexesjan/go-web-proxy-server/pkg/log"
)

func main() {
	port := 8080
	lc, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logpkg.Fatal(err)
	}
	log.ProxyListen("localhost", port)
	defer lc.Close()

	var cache sync.Map
	var blockList sync.Map

	go commandline.Dispatcher(&blockList)

	for {
		conn, err := lc.Accept()
		if err != nil {
			logpkg.Fatal(err)
		}

		go handleConnection(conn, &cache, &blockList)
	}
}

func handleConnection(conn net.Conn, cache, blockList *sync.Map) {
	defer conn.Close()

	timeStart := time.Now()
	req, err := http.NewRequest(conn)
	if err != nil {
		log.ProxyError(err)
		return
	}

	host := req.Headers["Host"]
	uri := fmt.Sprintf("%s%s", host, req.Path)
	// Handle website blocking.
	if blocked, ok := blockList.Load(host); ok {
		if blocked == true {
			log.ProxyBlock(host)
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
		log.ProxyHTTPSRequest(req.Method, host, req.HTTPVer)
		err := handleHTTPS(conn, host, req.HTTPVer)
		if err != nil {
			log.ProxyError(err)
		}
		return
	}

	// Handle HTTP request.
	cachedResponse, cacheFound := cache.Load(uri)
	if cacheFound {
		currTimeFormatted := time.Now().In(time.UTC).Format(http.TimeFormat)
		req.Headers["If-Modified-Since"] = currTimeFormatted
	}
	reqURL := fmt.Sprintf("http://%s%s", host, req.Path)
	resp, err := httpclient.Request(reqURL,
		&httpclient.Options{
			Method:  req.Method,
			HTTPVer: req.HTTPVer,
			Headers: req.Headers,
		},
	)
	if err != nil {
		log.ProxyError(err)
		return
	}
	duration := time.Since(timeStart)
	bandwidth := len(resp.String())
	// Cache is still valid. Return cached response.
	if cacheFound && resp.StatusCode == 304 {
		fmt.Fprint(conn, cachedResponse)
		log.ProxyHTTPResponse(
			req.Method,
			reqURL,
			req.HTTPVer,
			duration.String(),
			bandwidth,
			true,
		)
		return
	}

	// Forward response to client.
	fmt.Fprint(conn, resp)
	log.ProxyHTTPResponse(
		req.Method,
		reqURL,
		req.HTTPVer,
		duration.String(),
		bandwidth,
		false,
	)
	cache.Store(uri, resp.String())
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
