package http

import (
	"bufio"
	"io"
	"log"
	"strings"
)

// ReadHeaders will parse the HTTP headers in a HTTP message. The reader must
// have already read the HTTP status line prior to calling this function.
func ReadHeaders(reader *bufio.Reader) (headers map[string]string, err error) {
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
		headers[header[0]] = header[1]
	}

	return headers, err
}

// TimeFormat is an example of the HTTP date format. It can be used in the
// ParseTime function.
const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
