package http

import (
	"bufio"
	"io"
	"log"
	"strings"
)

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
		if trimmed == "" {
			break
		}

		header := strings.Split(trimmed, ": ")
		headers[header[0]] = header[1]
	}

	return headers, err
}
