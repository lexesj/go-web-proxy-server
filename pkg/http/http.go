package http

import (
	"bufio"
	"io"
	"log"
	"strings"
)

// Headers is string key value map of the HTTP headers
type Headers map[string]string

// CacheControl parses the Cache-Control header
func (headers *Headers) CacheControl() (cacheControl []string) {
	rawCacheControl, ok := (*headers)["Cache-Control"]
	cacheControl = []string{}
	if ok {
		cacheControl = strings.Split(rawCacheControl, ", ")
	}

	return cacheControl
}

// ReadHeaders will parse the HTTP headers in a HTTP message. The reader must
// have already read the HTTP status line prior to calling this function.
func ReadHeaders(reader *bufio.Reader) (headers Headers, err error) {
	headers = make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				return make(map[string]string), err
			}
			break
		}
		trimmed := strings.TrimRight(line, "\r\n")
		// End of headers, start of body.
		if trimmed == "" {
			break
		}
		header := strings.Split(trimmed, ": ")
		headers[standardiseHeaderKey(header[0])] = header[1]
	}

	return headers, err
}

func standardiseHeaderKey(headerKey string) (standarisedKey string) {
	standarisedKey = strings.Title(headerKey)
	return standarisedKey
}

// TimeFormat is an example of the HTTP date format. It can be used in the
// ParseTime function.
const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
