package log

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lexesjan/go-web-proxy-server/pkg/http"
	"github.com/mgutz/ansi"
)

// Bold ansi code
var Bold = ansi.ColorCode("white+bh")

// Prompt is the commandline prompt which has colours
var Prompt = fmt.Sprintf(
	"%s[%s%sProxy%s%s]$%s ",
	ansi.LightGreen,
	ansi.Reset,
	ansi.ColorCode("white+bh"),
	ansi.Reset,
	ansi.LightGreen,
	ansi.Reset,
)

type Logger struct {
	mu sync.Mutex
}

func NewLogger() (logger *Logger) {
	logger = &Logger{}
	return logger
}

func (logger *Logger) output(str string) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	currentTime := time.Now().Format("01/02/06 15:04:05")
	if !strings.HasSuffix(str, "\n") {
		str = str + "\n"
	}
	fmt.Fprintf(os.Stderr, "\r%s %s%s", currentTime, str, Prompt)
}

var logger = NewLogger()

// ProxyError logs an error has occurred and the message
func ProxyError(err error) {
	logger.output(fmt.Sprintf(
		"%s[%s%sError%s%s]%s [Message: %q]\n",
		ansi.LightRed,
		ansi.Reset,
		Bold,
		ansi.Reset,
		ansi.LightRed,
		ansi.Reset,
		err,
	))
}

// ProxyHTTPResponse logs a proxy HTTP response
func ProxyHTTPResponse(req *http.Request, resp *http.Response, time time.Duration, cached bool) {
	method := req.Method
	host := req.Headers["Host"]
	reqURL := fmt.Sprintf("http://%s%s", host, req.Path)
	httpVersion := resp.HTTPVer
	bandwidth := len(resp.String())
	proxy(
		"HTTP",
		"Response",
		ansi.LightBlue,
		fmt.Sprintf(
			"[Method: %q] [Request URL: %q] [HTTP Version: %q] [Bandwidth: %d bytes] [Time: %s]",
			method,
			reqURL,
			httpVersion,
			bandwidth,
			time,
		),
		cached,
	)
}

// ProxyHTTPSRequest logs a proxy HTTPS request
func ProxyHTTPSRequest(req *http.Request) {
	method := req.Method
	host := req.Headers["Host"]
	httpVersion := req.HTTPVer
	proxy(
		"HTTPS",
		"Request",
		ansi.Green,
		fmt.Sprintf(
			"[Method: %q] [Host: %q] [HTTP Version: %q]",
			method,
			host,
			httpVersion,
		),
		false,
	)
}

func proxy(protocol, messageType, colour, info string, cached bool) {
	cachedMessage := ""
	if cached {
		cachedMessage = fmt.Sprintf(
			"%s[%s%sCached%s%s]%s ",
			ansi.Yellow,
			ansi.Reset,
			Bold,
			ansi.Reset,
			ansi.Yellow,
			ansi.Reset,
		)
	}

	logger.output(fmt.Sprintf(
		"%s%s[%s%s%s %s%s%s]%s %s\n",
		cachedMessage,
		colour,
		ansi.Reset,
		Bold,
		protocol,
		messageType,
		ansi.Reset,
		colour,
		ansi.Reset,
		info,
	))
}

// ProxyBlock logs blocked message when the proxy blocks a website
func ProxyBlock(host string) {
	logger.output(fmt.Sprintf(
		"%s[%sBlocked%s]%s [Host %q]\n",
		ansi.Magenta,
		ansi.Reset,
		ansi.Magenta,
		ansi.Reset,
		host,
	))
}

// ProxyListen logs the listening message on proxy startup
func ProxyListen(host string, port int) {
	logger.output(fmt.Sprintf(
		"%s[%s%sListening on \"http://%s:%d\"%s%s]%s\n",
		ansi.LightYellow,
		ansi.Reset,
		Bold,
		host,
		port,
		ansi.Reset,
		ansi.LightYellow,
		ansi.Reset,
	))
}

// ProxyCacheStale logs a stale cache entry
func ProxyCacheStale(requestURL string) {
	logger.output(fmt.Sprintf("%s[%s%sCache Stale%s%s]%s [URI: %q]\n",
		ansi.LightRed,
		ansi.Reset,
		Bold,
		ansi.Reset,
		ansi.LightRed,
		ansi.Reset,
		requestURL,
	))
}
