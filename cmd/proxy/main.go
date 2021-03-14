package main

import (
	"fmt"
	"io"
	logpkg "log"
	"net"
	urlpkg "net/url"
	"strconv"
	"strings"
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

type cacheEntry struct {
	response string
	stale    bool
}

func (entry *cacheEntry) resetTimer(uri string, cacheControl []string) (err error) {
	maxAge := 0
	for _, elem := range cacheControl {
		if strings.HasPrefix(elem, "max-age") {
			tokens := strings.Split(elem, "=")
			maxAge, err = strconv.Atoi(tokens[1])
			if err != nil {
				return err
			}
		}
	}

	// Mark expired cache as stale
	time.AfterFunc(time.Duration(maxAge)*time.Second, func() {
		entry.stale = true
		log.ProxyCacheStale(uri)
	})

	return nil
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
	reqOptions := &httpclient.Options{
		Method:  req.Method,
		HTTPVer: req.HTTPVer,
		Headers: req.Headers,
	}
	reqURL := fmt.Sprintf("http://%s%s", host, req.Path)
	cachedEntryInterface, cacheFound := cache.Load(uri)
	if cacheFound {
		cachedEntry := cachedEntryInterface.(*cacheEntry)
		if cachedEntry.stale {
			currTimeFormatted := time.Now().In(time.UTC).Format(http.TimeFormat)
			req.Headers["If-Modified-Since"] = currTimeFormatted
			resp, err := httpclient.Request(reqURL, reqOptions)
			if err != nil {
				log.ProxyError(err)
				return
			}

			// Cached response is still valid
			if resp.StatusCode == 304 {
				fmt.Fprint(conn, cachedEntry.response)
				bandwidth := len(resp.String())
				duration := time.Since(timeStart)
				log.ProxyHTTPResponse(
					req.Method,
					reqURL,
					req.HTTPVer,
					duration.String(),
					bandwidth,
					true,
				)
				cachedEntry.resetTimer(uri, resp.Headers.CacheControl())
				if err != nil {
					log.ProxyError(err)
				}
				return
			}

			// Forward response to client.
			fmt.Fprint(conn, resp)
			bandwidth := len(resp.String())
			duration := time.Since(timeStart)
			log.ProxyHTTPResponse(
				req.Method,
				reqURL,
				req.HTTPVer,
				duration.String(),
				bandwidth,
				false,
			)
			err = cacheResponse(uri, resp, cache)
			if err != nil {
				log.ProxyError(err)
			}
		} else {
			// Return cached response as it is not stale
			fmt.Fprint(conn, cachedEntry.response)
			duration := time.Since(timeStart)
			log.ProxyHTTPResponse(
				req.Method,
				reqURL,
				req.HTTPVer,
				duration.String(),
				0,
				true,
			)
		}
		return
	}

	// Response not in cache
	resp, err := httpclient.Request(reqURL, reqOptions)
	if err != nil {
		log.ProxyError(err)
		return
	}

	// Forward response to client.
	fmt.Fprint(conn, resp)
	bandwidth := len(resp.String())
	duration := time.Since(timeStart)
	log.ProxyHTTPResponse(
		req.Method,
		reqURL,
		req.HTTPVer,
		duration.String(),
		bandwidth,
		false,
	)
	err = cacheResponse(uri, resp, cache)
	if err != nil {
		log.ProxyError(err)
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

func cacheResponse(uri string, resp *http.Response, cache *sync.Map) (err error) {
	contains := func(arr []string, str string) bool {
		for _, elem := range arr {
			if elem == str {
				return true
			}
		}
		return false
	}

	newCacheEntry := &cacheEntry{
		response: resp.String(),
		stale:    false,
	}
	cacheControl := resp.Headers.CacheControl()
	// Should be cached
	if !contains(cacheControl, "no-store") {
		cache.Store(uri, newCacheEntry)
		err = newCacheEntry.resetTimer(uri, cacheControl)
		if err != nil {
			return err
		}
	}

	return nil
}
