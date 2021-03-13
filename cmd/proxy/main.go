package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	urlpkg "net/url"
	"os"
	"strconv"
	"strings"
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
	log.Printf("[Listning on \"http://localhost:%d\"]\n", port)
	defer lc.Close()

	var cache sync.Map
	var blockList sync.Map

	go commandDispatcher(&blockList)

	for {
		conn, err := lc.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(conn, &cache, &blockList)
	}
}

func commandDispatcher(blockList *sync.Map) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("[Proxy]$ ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("[Error] [Message: %q]\n", err)
		}

		trimmedInput := strings.TrimRight(input, "\n")
		tokens := strings.Split(trimmedInput, " ")
		command := tokens[0]
		if command != "" {
			switch command {
			case "block":
				if len(tokens) == 1 {
					fmt.Fprintf(os.Stderr, "usage: block <domain name>\n")
					continue
				}

				website := tokens[1]
				_, loaded := blockList.LoadOrStore(website, true)
				if !loaded {
					fmt.Printf("%s: blocked %q\n", command, website)
				} else {
					fmt.Fprintf(os.Stderr, "%s: website %q already blocked\n", command, website)
				}
			case "unblock":
				if len(tokens) == 1 {
					fmt.Fprintf(os.Stderr, "usage: unblock <domain name>\n")
				}

				website := tokens[1]
				_, found := blockList.LoadAndDelete(website)
				if found {
					fmt.Printf("%s: unblocked %q\n", command, website)
				} else {
					fmt.Fprintf(os.Stderr, "%s: website %q not blocked\n", command, website)
				}
			default:
				fmt.Printf("proxy: %q: command not found\n", tokens[0])
			}
		}
	}
}

func handleConnection(conn net.Conn, cache, blockList *sync.Map) {
	defer conn.Close()

	timeStart := time.Now()
	req, err := http.NewRequest(conn)
	if err != nil {
		log.Printf("[Error] [Message: %q]\n", err)
		return
	}

	host := req.Headers["Host"]
	uri := fmt.Sprintf("%s%s", host, req.Path)
	// Handle website blocking.
	if blocked, ok := blockList.Load(host); ok {
		if blocked == true {
			log.Printf("[Blocked] [Host: %q]\n", host)
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
		log.Printf("[HTTPS Request] [Method: %q] [Host: %q] [HTTP Version: %q]\n",
			req.Method, host, req.HTTPVer)
		err := handleHTTPS(conn, host, req.HTTPVer)
		if err != nil {
			log.Printf("[Error] [Message: %q]\n", err)
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
		log.Printf("[Error] [Message: %q]\n", err)
		return
	}
	duration := time.Since(timeStart)
	bandwidth := len(resp.String())
	// Cache is still valid. Return cached response.
	if cacheFound && resp.StatusCode == 304 {
		log.Printf("[Cached] [HTTP Response] [Method: %q] [Request URL: %q] [HTTP Version: %q] [Bandwidth: %d bytes] [Time: %s]\n",
			req.Method, reqURL, req.HTTPVer, bandwidth, duration)
		fmt.Fprint(conn, cachedResponse)
		return
	}

	// Forward response to client.
	log.Printf("[HTTP Response] [Method: %q] [Request URL: %q] [HTTP Version: %q] [Bandwidth: %d bytes] [Time: %s]\n",
		req.Method, reqURL, req.HTTPVer, bandwidth, duration)
	fmt.Fprint(conn, resp)
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
