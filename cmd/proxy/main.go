package main

import (
	"fmt"
	"io"
	logpkg "log"
	"net"
	urlpkg "net/url"
	"os"
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
	err = handleHTTP(conn, req, cache)
	if err != nil {
		log.ProxyError(err)
	}
}

func handleHTTPS(conn net.Conn, req *http.Request) (err error) {
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

	log.ProxyHTTPSRequest(req)
	return nil
}

func handleHTTP(conn net.Conn, req *http.Request, cache *sync.Map) (err error) {
	host := req.Headers["Host"]
	timeStart := time.Now()
	uri := fmt.Sprintf("%s%s", host, req.Path)
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
		} else {
			// Return cached response as it is not stale
			fmt.Fprint(conn, cachedEntry.response)
			duration := time.Since(timeStart)
			log.ProxyHTTPResponse(req, &http.Response{}, duration, true)
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
		cachedEntry := cachedEntryInterface.(*cacheEntry)
		fmt.Fprint(conn, cachedEntry.response)
		duration := time.Since(timeStart)
		cachedEntry.resetTimer(uri, resp.Headers.CacheControl())
		if err != nil {
			return err
		}
		log.ProxyHTTPResponse(req, resp, duration, true)
		return nil
	}

	// Forward response to client.
	fmt.Fprint(conn, resp)
	duration := time.Since(timeStart)
	err = cacheResponse(uri, resp, cache)
	if err != nil {
		return err
	}
	log.ProxyHTTPResponse(req, resp, duration, false)

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
	uncacheable := contains(cacheControl, "no-store") || resp.StatusCode == 304
	// Can't be cached.
	if uncacheable {
		return nil
	}

	cache.Store(uri, newCacheEntry)
	err = newCacheEntry.resetTimer(uri, cacheControl)
	if err != nil {
		return err
	}

	return nil
}
