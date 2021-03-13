package httpclient

import (
	"fmt"
	"net"
	urlpkg "net/url"

	"github.com/lexesjan/go-web-proxy-server/pkg/http"
)

// Options represent the options that a request will take
type Options struct {
	Method  string
	Headers map[string]string
	HTTPVer string
}

func Request(rawurl string, options *Options) (resp *http.Response, err error) {
	url, err := urlpkg.Parse(rawurl)
	if url.Host == "" {
		return &http.Response{}, fmt.Errorf("%q is not a valid URL\n", rawurl)
	}
	if err != nil {
		return &http.Response{}, err
	}

	host := url.Hostname()
	path := url.Path
	if path == "" {
		path = "/"
	}
	port := url.Port()
	if port == "" {
		port = "80"
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return &http.Response{}, err
	}
	defer conn.Close()

	if _, ok := options.Headers["Host"]; !ok {
		options.Headers["Host"] = host
	}

	req := &http.Request{
		Method:  options.Method,
		Path:    path,
		HTTPVer: options.HTTPVer,
		Headers: options.Headers,
	}
	fmt.Fprint(conn, req)

	return http.NewResponse(conn)
}
