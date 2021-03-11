package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	urlpkg "net/url"
	"strings"
)

func main() {
	port := 8080
	lc, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Listning on http://localhost:%d\n", port)
	defer lc.Close()

	for {
		conn, err := lc.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(conn)
	}
}

func readStatus(reader *bufio.Reader) (string, string, string, error) {
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return "", "", "", err
	}

	status := strings.Split(statusLine, " ")
	method := status[0]
	httpVer := strings.TrimRight(status[2], "\r\n")

	url, err := urlpkg.Parse(status[1])
	if err != nil {
		return "", "", "", err
	}

	return method, url.Path, httpVer, nil
}

func readHostHeader(reader *bufio.Reader) (string, string, error) {
	hostHeader, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	trimmedHostHeader := strings.TrimRight(hostHeader, "\r\n")
	host := strings.Split(trimmedHostHeader, ": ")
	hostPortPair := strings.Split(host[1], ":")

	if len(hostPortPair) == 1 {
		return hostPortPair[0], "", nil
	}

	return hostPortPair[0], hostPortPair[1], nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	_, _, _, err := readStatus(reader)

	if err != nil {
		log.Println(err)
	}

	host, port, err := readHostHeader(reader)

	if err != nil {
		log.Println(err)
	}

	fmt.Println(host, port)
}
