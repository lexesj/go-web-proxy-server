package log

import (
	"fmt"
	logpkg "log"

	"github.com/mgutz/ansi"
)

var bold = ansi.ColorCode("white+bh")

// ProxyError logs an error has occurred and the message
func ProxyError(err error) {
	logpkg.Printf(
		"%s[%s%sError%s%s]%s [Message: %q]\n",
		ansi.LightRed,
		ansi.Reset,
		bold,
		ansi.Reset,
		ansi.LightRed,
		ansi.Reset,
		err,
	)
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
		ansi.LightCyan,
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
	cachedMessage := "[Cached] "
	if !cached {
		cachedMessage = ""
	}

	logpkg.Printf(
		"%s%s[%s%s%s %s%s%s]%s %s\n",
		cachedMessage,
		colour,
		ansi.Reset,
		bold,
		protocol,
		messageType,
		ansi.Reset,
		colour,
		ansi.Reset,
		info,
	)
}

// ProxyBlock logs blocked message when the proxy blocks a website
func ProxyBlock(host string) {
	logpkg.Printf(
		"%s[%sBlocked%s]%s [Host %q]\n",
		ansi.Magenta,
		ansi.Reset,
		ansi.Magenta,
		ansi.Reset,
		host,
	)
}

// ProxyListen logs the listening message on proxy startup
func ProxyListen(host string, port int) {
	logpkg.Printf(
		"%s[%s%sListening on \"http://%s:%d\"%s%s]%s\n",
		ansi.LightYellow,
		ansi.Reset,
		bold,
		host,
		port,
		ansi.Reset,
		ansi.LightYellow,
		ansi.Reset,
	)
}
