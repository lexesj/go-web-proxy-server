package log

import (
	"fmt"
	"os"
	"strings"
	"time"

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

func output(str string) {
	currentTime := time.Now().Format("01/02/06 15:04:05")
	if !strings.HasSuffix(str, "\n") {
		str = str + "\n"
	}
	fmt.Fprintf(os.Stderr, "\r%s %s%s", currentTime, str, Prompt)
}

// ProxyError logs an error has occurred and the message
func ProxyError(err error) {
	output(fmt.Sprintf(
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
func ProxyHTTPResponse(method, requestURL, httpVersion, time string, bandwidth int, cached bool) {
	proxy(
		"HTTP",
		"Response",
		ansi.LightBlue,
		fmt.Sprintf(
			"[Method: %q] [Request URL: %q] [HTTP Version: %q] [Bandwidth: %d bytes] [Time: %s]",
			method,
			requestURL,
			httpVersion,
			bandwidth,
			time,
		),
		cached,
	)
}

// ProxyHTTPSRequest logs a proxy HTTPS request
func ProxyHTTPSRequest(method, host, httpVersion string) {
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

	output(fmt.Sprintf(
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
	output(fmt.Sprintf(
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
	output(fmt.Sprintf(
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
	output(fmt.Sprintf("%s[%s%sCache Stale%s%s]%s [URI: %q]\n",
		ansi.LightRed,
		ansi.Reset,
		Bold,
		ansi.Reset,
		ansi.LightRed,
		ansi.Reset,
		requestURL,
	))
}
