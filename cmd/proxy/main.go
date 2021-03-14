package main

import (
	"fmt"
	"io"
	logpkg "log"
	"net"
	urlpkg "net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lexesjan/go-web-proxy-server/pkg/cache"
	"github.com/lexesjan/go-web-proxy-server/pkg/commandline"
	"github.com/lexesjan/go-web-proxy-server/pkg/http"
	"github.com/lexesjan/go-web-proxy-server/pkg/httpclient"
	"github.com/lexesjan/go-web-proxy-server/pkg/log"
	"github.com/lexesjan/go-web-proxy-server/pkg/metrics"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <port number>\n", args[0])
		return
	}

	port, err := strconv.Atoi(args[1])
	if err != nil || port < 0 && port > 65535 {
		fmt.Fprintf(os.Stderr, "error: %s is not a valid port number\n", err)
		return
	}

	lc, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logpkg.Fatal(err)
	}
	defer lc.Close()
	log.ProxyListen("localhost", port)

	cache := cache.NewCache()
	var blockList sync.Map
	metrics := metrics.NewMetrics()

	go commandline.Dispatcher(&blockList, metrics)

	for {
		conn, err := lc.Accept()
		if err != nil {
			logpkg.Fatal(err)
		}

		go handleConnection(conn, cache, &blockList, metrics)
	}
}

func handleConnection(conn net.Conn, cache *cache.Cache, blockList *sync.Map, metrics *metrics.Metrics) {
	defer conn.Close()

	req, err := http.NewRequest(conn)
	if err != nil {
		log.ProxyError(err)
		return
	}

	host := req.Headers["Host"]
	// Handle website blocking.
	if blocked, ok := blockList.Load(host); ok {
		if blocked == true {
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
			log.ProxyBlock(host)
			return
		}
	}

	// Handle HTTPS request.
	if req.Method == "CONNECT" {
		err := handleHTTPS(conn, req)
		if err != nil {
			log.ProxyError(err)
		}
		return
	}

	// Handle HTTP request.
	err = handleHTTP(conn, req, cache, metrics)
	if err != nil {
		log.ProxyError(err)
	}
}

func handleHTTPS(conn net.Conn, req *http.Request) (err error) {
	log.ProxyHTTPSRequest(req)
	rawurl := req.Headers["Host"]
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

func handleHTTP(conn net.Conn, req *http.Request, cache *cache.Cache, metrics *metrics.Metrics) (err error) {
	startTime := time.Now()
	host := req.Headers["Host"]
	reqOptions := &httpclient.Options{
		Method:  req.Method,
		HTTPVer: req.HTTPVer,
		Headers: req.Headers,
	}
	reqURL := fmt.Sprintf("http://%s%s", host, req.Path)
	cachedEntry, cacheFound := cache.Get(reqURL)
	if cacheFound {
		if cachedEntry.Stale {
			currTimeFormatted := time.Now().In(time.UTC).Format(http.TimeFormat)
			req.Headers["If-Modified-Since"] = currTimeFormatted
		} else {
			// Return cached response as it is not stale
			fmt.Fprint(conn, cachedEntry.Response)
			duration := time.Since(startTime)
			log.ProxyHTTPResponse(req, &http.Response{}, duration, true)
			metrics.AddMetrics(reqURL, cachedEntry, duration, 0)
			return
		}
	}

	// Response not in cache or validate cache
	resp, err := httpclient.Request(reqURL, reqOptions)
	if err != nil {
		return err
	}

	// Cached response is still valid
	if cacheFound && resp.StatusCode == 304 {
		fmt.Fprint(conn, cachedEntry.Response)
		duration := time.Since(startTime)
		cachedEntry.ResetTimer(reqURL, resp.Headers.CacheControl())
		if err != nil {
			return err
		}
		log.ProxyHTTPResponse(req, resp, duration, true)
		metrics.AddMetrics(reqURL, cachedEntry, duration, int64(len(resp.String())))
		return nil
	}

	// Forward response to client.
	fmt.Fprint(conn, resp)
	duration := time.Since(startTime)
	err = cache.CacheResponse(reqURL, resp, duration)
	if err != nil {
		return err
	}
	log.ProxyHTTPResponse(req, resp, duration, false)

	return nil
}
