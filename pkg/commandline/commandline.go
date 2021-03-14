package commandline

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lexesjan/go-web-proxy-server/pkg/log"
	"github.com/lexesjan/go-web-proxy-server/pkg/metrics"
)

// Dispatcher handles the user input
func Dispatcher(blockList *sync.Map, metrics *metrics.Metrics) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\r%s", log.Prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			log.ProxyError(err)
		}

		trimmedInput := strings.TrimRight(input, "\n")
		tokens := strings.Split(trimmedInput, " ")
		command := tokens[0]
		if command != "" {
			switch command {
			case "block":
				if len(tokens) == 1 || len(tokens) > 2 {
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
				if len(tokens) == 1 || len(tokens) > 2 {
					fmt.Fprintf(os.Stderr, "usage: unblock <domain name>\n")
					continue
				}

				website := tokens[1]
				_, found := blockList.LoadAndDelete(website)
				if found {
					fmt.Printf("%s: unblocked %q\n", command, website)
				} else {
					fmt.Fprintf(os.Stderr, "%s: website %q not blocked\n", command, website)
				}
			case "metrics":
				if len(tokens) > 1 {
					fmt.Fprintf(os.Stderr, "usage: metrics\n")
					continue
				}

				fmt.Println(metrics)
			case "clear":
				if len(tokens) > 1 {
					fmt.Fprintf(os.Stderr, "usage: clear\n")
					continue
				}

				// Move cursor to (0,0) and clear console
				fmt.Print("\033[0;0H")
				fmt.Print("\033[2J")
			default:
				fmt.Printf("proxy: %q: command not found\n", tokens[0])
			}
		}
	}
}
